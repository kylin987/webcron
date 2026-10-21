package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"time"
)

const (
	TASK_SUCCESS = 0  // 任务执行成功
	TASK_ERROR   = -1 // 任务执行出错
	TASK_TIMEOUT = -2 // 任务执行超时
)

type Task struct {
	Tags         []*TaskTag `orm:"reverse(many)"`
	Id           int
	UserId       int
	GroupId      int
	TaskName     string
	TaskType     int
	Description  string
	CronSpec     string
	Concurrent   int
	Command      string
	Status       int
	Notify       int
	NotifyEmail  string
	Timeout      int
	ExecuteTimes int
	PrevTime     int64
	CreateTime   int64
	LastLogId    int64
	LastStatus   int
	LastRunTime  int64
	LastError    string `orm:"size(300)"`
}

func (t *Task) TableName() string {
	return TableName("task")
}

func (t *Task) Update(fields ...string) error {
	// Latest execution fields are owned by TaskLogAdd, never by scheduler snapshots.
	if len(fields) == 0 {
		fields = []string{"UserId", "GroupId", "TaskName", "TaskType", "Description", "CronSpec", "Concurrent", "Command", "Status", "Notify", "NotifyEmail", "Timeout", "ExecuteTimes", "PrevTime", "CreateTime"}
	}
	if _, err := orm.NewOrm().Update(t, fields...); err != nil {
		return err
	}
	return nil
}

func TaskAdd(task *Task) (int64, error) {
	if task.TaskName == "" {
		return 0, fmt.Errorf("TaskName字段不能为空")
	}
	if task.CronSpec == "" {
		return 0, fmt.Errorf("CronSpec字段不能为空")
	}
	if task.Command == "" {
		return 0, fmt.Errorf("Command字段不能为空")
	}
	if task.CreateTime == 0 {
		task.CreateTime = time.Now().Unix()
	}
	return orm.NewOrm().Insert(task)
}

func TaskGetList(page, pageSize int, filters ...interface{}) ([]*Task, int64) {
	offset := (page - 1) * pageSize

	tasks := make([]*Task, 0)

	query := orm.NewOrm().QueryTable(TableName("task"))
	if len(filters) > 0 {
		l := len(filters)
		for k := 0; k < l; k += 2 {
			query = query.Filter(filters[k].(string), filters[k+1])
		}
	}
	total, _ := query.Count()
	query.OrderBy("-id").Limit(pageSize, offset).All(&tasks)

	return tasks, total
}

func TaskResetGroupId(groupId int) (int64, error) {
	return orm.NewOrm().QueryTable(TableName("task")).Filter("group_id", groupId).Update(orm.Params{
		"group_id": 0,
	})
}

func TaskGetById(id int) (*Task, error) {
	task := &Task{
		Id: id,
	}

	err := orm.NewOrm().Read(task)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func TaskDel(id int) error {
	o := orm.NewOrm()
	if err := o.Begin(); err != nil {
		return err
	}
	defer o.Rollback()
	if err := o.ReadForUpdate(&Task{Id: id}); err != nil {
		return err
	}
	if _, err := o.QueryTable(TableName("task_tag")).Filter("task_id", id).Delete(); err != nil {
		return err
	}
	if _, err := o.QueryTable(TableName("task")).Filter("id", id).Delete(); err != nil {
		return err
	}
	return o.Commit()
}
