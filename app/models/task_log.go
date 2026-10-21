package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
)

type TaskLog struct {
	Id          int
	TaskId      int
	Output      string
	Error       string
	Status      int
	ProcessTime int
	CreateTime  int64
}

func (t *TaskLog) TableName() string {
	return TableName("task_log")
}

func TaskLogAdd(t *TaskLog) (int64, error) {
	o := orm.NewOrm()
	if err := o.Begin(); err != nil {
		return 0, err
	}
	defer o.Rollback()
	id, err := o.Insert(t)
	if err != nil {
		return 0, err
	}
	// Preserve completion order even when concurrent executions commit out of order.
	query := fmt.Sprintf("UPDATE `%s` SET last_log_id=?,last_status=?,last_run_time=?,last_error=LEFT(?,300) WHERE id=? AND last_log_id<?", TableName("task"))
	if _, err = o.Raw(query, id, t.Status, t.CreateTime, t.Error, t.TaskId, id).Exec(); err != nil {
		return 0, err
	}
	if err = o.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func TaskLogGetList(page, pageSize int, filters ...interface{}) ([]*TaskLog, int64) {
	offset := (page - 1) * pageSize

	logs := make([]*TaskLog, 0)

	query := orm.NewOrm().QueryTable(TableName("task_log"))
	if len(filters) > 0 {
		l := len(filters)
		for k := 0; k < l; k += 2 {
			query = query.Filter(filters[k].(string), filters[k+1])
		}
	}

	total, _ := query.Count()
	query.OrderBy("-id").Limit(pageSize, offset).All(&logs)

	return logs, total
}

func TaskLogGetById(id int) (*TaskLog, error) {
	obj := &TaskLog{
		Id: id,
	}

	err := orm.NewOrm().Read(obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func TaskLogDelById(id int) error {
	_, err := orm.NewOrm().QueryTable(TableName("task_log")).Filter("id", id).Delete()
	return err
}

func TaskLogDelByTaskId(taskId int) (int64, error) {
	return orm.NewOrm().QueryTable(TableName("task_log")).Filter("task_id", taskId).Delete()
}

// TaskLogDelByTime 删除指定时间戳之前的任务执行记录，返回删除行数。
func TaskLogDelByTime(before int64) (int64, error) {
	return orm.NewOrm().QueryTable(TableName("task_log")).Filter("create_time__lt", before).Delete()
}
