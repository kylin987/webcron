package models

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/astaxie/beego/orm"
)

// TaskTag stores one exact label per task. Labels share the task list's visibility.
type TaskTag struct {
	Id   int
	Task *Task  `orm:"rel(fk);column(task_id)"`
	Name string `orm:"size(30)"`
}

func (t *TaskTag) TableName() string { return TableName("task_tag") }

func ParseTaskTags(input string) ([]string, error) {
	names := make([]string, 0)
	seen := make(map[string]bool)
	for _, name := range strings.FieldsFunc(input, func(r rune) bool { return r == ',' || r == '，' || r == '\n' || r == '\r' }) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		if utf8.RuneCountInString(name) > 30 {
			return nil, fmt.Errorf("每个标签不能超过30个字")
		}
		names = append(names, name)
		seen[name] = true
	}
	if len(names) > 20 {
		return nil, fmt.Errorf("每个任务最多设置20个标签")
	}
	return names, nil
}

// TaskSaveWithTags atomically saves task configuration and its labels.
func TaskSaveWithTags(task *Task, names []string) error {
	if task.TaskName == "" || task.CronSpec == "" || task.Command == "" {
		return fmt.Errorf("请填写完整信息")
	}
	o := orm.NewOrm()
	if err := o.Begin(); err != nil {
		return err
	}
	defer o.Rollback()
	if task.Id == 0 {
		if task.CreateTime == 0 {
			task.CreateTime = time.Now().Unix()
		}
		if _, err := o.Insert(task); err != nil {
			return err
		}
	} else {
		// Serialize concurrent edits and avoid overwriting scheduler-owned counters/status.
		if err := o.ReadForUpdate(&Task{Id: task.Id}); err != nil {
			return err
		}
		if _, err := o.Update(task, "TaskName", "Description", "GroupId", "Concurrent", "CronSpec", "Command", "Notify", "NotifyEmail", "Timeout"); err != nil {
			return err
		}
	}
	if _, err := o.QueryTable(TableName("task_tag")).Filter("task_id", task.Id).Delete(); err != nil {
		return err
	}
	for _, name := range names {
		if _, err := o.Insert(&TaskTag{Task: task, Name: name}); err != nil {
			return err
		}
	}
	return o.Commit()
}

func TaskTagsByTaskIDs(ids []int) (map[int][]string, error) {
	result := make(map[int][]string)
	if len(ids) == 0 {
		return result, nil
	}
	var tags []*TaskTag
	_, err := orm.NewOrm().QueryTable(TableName("task_tag")).Filter("task_id__in", ids).OrderBy("id").Limit(-1).All(&tags)
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		result[tag.Task.Id] = append(result[tag.Task.Id], tag.Name)
	}
	return result, nil
}

func TaskTagNames() ([]string, error) {
	var rows []orm.Params
	_, err := orm.NewOrm().QueryTable(TableName("task_tag")).OrderBy("name").Distinct().Limit(-1).Values(&rows, "Name")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row["Name"].(string))
	}
	return names, nil
}
