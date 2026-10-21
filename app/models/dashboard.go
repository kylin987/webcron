package models

import (
	"fmt"
	"sort"

	"github.com/astaxie/beego/orm"
)

// DashboardLog contains only the fields rendered on the overview, never full output.
type DashboardLog struct {
	Id          int
	TaskId      int
	TaskName    string
	Status      int
	CreateTime  int64
	ProcessTime int
	OutputSize  int64
	Output      string
	Error       string
}

func DashboardLogs(failed bool) ([]DashboardLog, error) {
	o := orm.NewOrm()
	base := fmt.Sprintf("SELECT l.id, l.task_id, COALESCE(t.task_name, '任务已删除') AS task_name, l.status, l.create_time, l.process_time, OCTET_LENGTH(l.output) AS output_size, LEFT(l.output, 100) AS output, LEFT(l.error, 300) AS error FROM `%s` l LEFT JOIN `%s` t ON t.id=l.task_id", TableName("task_log"), TableName("task"))
	logs := make([]DashboardLog, 0, 40)
	if !failed {
		_, err := o.Raw(base + " ORDER BY l.id DESC LIMIT 20").QueryRows(&logs)
		return logs, err
	}
	// Equality on each status uses (status,id), without sorting/scanning all successes.
	for _, status := range []int{TASK_ERROR, TASK_TIMEOUT} {
		var rows []DashboardLog
		if _, err := o.Raw(base+" WHERE l.status=? ORDER BY l.id DESC LIMIT 20", status).QueryRows(&rows); err != nil {
			return nil, err
		}
		logs = append(logs, rows...)
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].Id > logs[j].Id })
	if len(logs) > 20 {
		logs = logs[:20]
	}
	return logs, nil
}

type FailedTaskSummary struct {
	Count int64        `json:"count"`
	Tasks []FailedTask `json:"tasks"`
}

type FailedTask struct {
	Id          int    `json:"id"`
	TaskName    string `json:"task_name"`
	LastStatus  int    `json:"last_status"`
	LastRunTime int64  `json:"last_run_time"`
	LastError   string `json:"last_error"`
}

func CurrentFailedTasks() (*FailedTaskSummary, error) {
	query := orm.NewOrm().QueryTable(TableName("task")).Filter("last_status__lt", 0)
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	result := &FailedTaskSummary{Count: count, Tasks: make([]FailedTask, 0)}
	// Query into Task since the QueryTable model defines its field mapping.
	var tasks []*Task
	if _, err = query.OrderBy("-last_log_id").Limit(5).All(&tasks, "Id", "TaskName", "LastStatus", "LastRunTime", "LastError"); err != nil {
		return nil, err
	}
	for _, t := range tasks {
		result.Tasks = append(result.Tasks, FailedTask{t.Id, t.TaskName, t.LastStatus, t.LastRunTime, t.LastError})
	}
	return result, nil
}
