package controllers

import (
	"github.com/astaxie/beego"
	libcron "github.com/lisijie/cron"
	"github.com/lisijie/webcron/app/jobs"
	"github.com/lisijie/webcron/app/libs"
	"github.com/lisijie/webcron/app/models"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TaskController struct {
	BaseController
}

// 任务列表
func (this *TaskController) List() {
	page, _ := this.GetInt("page")
	if page < 1 {
		page = 1
	}
	groupId, _ := this.GetInt("groupid")
	keyword := strings.TrimSpace(this.GetString("keyword"))
	tag := strings.TrimSpace(this.GetString("tag"))
	resultFilter := this.GetString("result")
	switch resultFilter {
	case "failed", "success", "timeout":
	default:
		resultFilter = ""
	}

	filters := make([]interface{}, 0)
	if groupId > 0 {
		filters = append(filters, "group_id", groupId)
	}
	if keyword != "" {
		filters = append(filters, "task_name__contains", keyword)
	}
	if tag != "" {
		filters = append(filters, "Tags__Name", tag)
	}
	switch resultFilter {
	case "failed":
		filters = append(filters, "last_status__lt", 0)
	case "success":
		filters = append(filters, "last_log_id__gt", 0, "last_status", models.TASK_SUCCESS)
	case "timeout":
		filters = append(filters, "last_status", models.TASK_TIMEOUT)
	}
	result, count := models.TaskGetList(page, this.pageSize, filters...)
	taskIDs := make([]int, 0, len(result))
	for _, task := range result {
		taskIDs = append(taskIDs, task.Id)
	}
	taskTags, err := models.TaskTagsByTaskIDs(taskIDs)
	if err != nil {
		this.showMsg("读取任务标签失败：" + err.Error())
	}
	tagNames, err := models.TaskTagNames()
	if err != nil {
		this.showMsg("读取标签列表失败：" + err.Error())
	}

	// 分组列表
	groups, _ := models.TaskGroupGetList(1, 100)

	groupIds := make([]int, 0)
	groupIdExists := make(map[int]bool)
	for _, task := range result {
		if task.GroupId > 0 && !groupIdExists[task.GroupId] {
			groupIds = append(groupIds, task.GroupId)
			groupIdExists[task.GroupId] = true
		}
	}

	taskGroups, _ := models.TaskGroupGetByIds(groupIds)
	groupNames := make(map[int]string, len(taskGroups))
	for _, group := range taskGroups {
		groupNames[group.Id] = group.GroupName
	}

	list := make([]map[string]interface{}, len(result))
	for k, v := range result {
		row := make(map[string]interface{})
		row["id"] = v.Id
		row["group_name"] = groupNames[v.GroupId]
		row["name"] = v.TaskName
		row["command"] = v.Command
		row["tags"] = taskTags[v.Id]
		row["last_log_id"] = v.LastLogId
		row["last_status"] = v.LastStatus
		row["last_error"] = v.LastError
		row["last_run_time"] = ""
		if v.LastLogId > 0 {
			row["last_run_time"] = beego.Date(time.Unix(v.LastRunTime, 0), "Y-m-d H:i:s")
		}
		row["cron_spec"] = v.CronSpec
		row["status"] = v.Status
		row["description"] = v.Description

		e := jobs.GetEntryById(v.Id)
		if e != nil {
			row["next_time"] = beego.Date(e.Next, "Y-m-d H:i:s")
			row["prev_time"] = "-"
			if e.Prev.Unix() > 0 {
				row["prev_time"] = beego.Date(e.Prev, "Y-m-d H:i:s")
			} else if v.PrevTime > 0 {
				row["prev_time"] = beego.Date(time.Unix(v.PrevTime, 0), "Y-m-d H:i:s")
			}
			row["running"] = 1
		} else {
			row["next_time"] = "-"
			if v.PrevTime > 0 {
				row["prev_time"] = beego.Date(time.Unix(v.PrevTime, 0), "Y-m-d H:i:s")
			} else {
				row["prev_time"] = "-"
			}
			row["running"] = 0
		}
		list[k] = row
	}

	this.Data["pageTitle"] = "任务列表"
	this.Data["list"] = list
	this.Data["groups"] = groups
	this.Data["groupid"] = groupId
	this.Data["keyword"] = keyword
	this.Data["tag"] = tag
	this.Data["result"] = resultFilter
	this.Data["tagNames"] = tagNames
	this.Data["wideLayout"] = true
	query := url.Values{"groupid": {strconv.Itoa(groupId)}, "keyword": {keyword}, "tag": {tag}, "result": {resultFilter}}
	this.Data["pageBar"] = libs.NewPager(page, int(count), this.pageSize, beego.URLFor("TaskController.List")+"?"+query.Encode(), true).ToString()
	this.display()
}

// 添加任务
func (this *TaskController) Add() {

	if this.isPost() {
		task := new(models.Task)
		task.UserId = this.userId
		task.GroupId, _ = this.GetInt("group_id")
		task.TaskName = strings.TrimSpace(this.GetString("task_name"))
		task.Description = strings.TrimSpace(this.GetString("description"))
		tags, err := models.ParseTaskTags(this.GetString("tags"))
		if err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		task.Concurrent, _ = this.GetInt("concurrent")
		task.CronSpec = strings.TrimSpace(this.GetString("cron_spec"))
		task.Command = strings.TrimSpace(this.GetString("command"))
		task.Notify, _ = this.GetInt("notify")
		task.Timeout, _ = this.GetInt("timeout")

		notifyEmail := strings.TrimSpace(this.GetString("notify_email"))
		if notifyEmail != "" {
			emailList := make([]string, 0)
			tmp := strings.Split(notifyEmail, "\n")
			for _, v := range tmp {
				v = strings.TrimSpace(v)
				if !libs.IsEmail([]byte(v)) {
					this.ajaxMsg("无效的Email地址："+v, MSG_ERR)
				} else {
					emailList = append(emailList, v)
				}
			}
			task.NotifyEmail = strings.Join(emailList, "\n")
		}

		if task.TaskName == "" || task.CronSpec == "" || task.Command == "" {
			this.ajaxMsg("请填写完整信息", MSG_ERR)
		}
		if _, err := libcron.Parse(task.CronSpec); err != nil {
			this.ajaxMsg("cron表达式无效", MSG_ERR)
		}
		err = models.TaskSaveWithTags(task, tags)
		if err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		this.audit("task_create", "task", task.Id, models.OP_RESULT_OK, map[string]interface{}{"task": taskSnapshot(task), "tags": tags})

		this.ajaxMsg("", MSG_OK)
	}

	// 分组列表
	groups, _ := models.TaskGroupGetList(1, 100)
	tagNames, err := models.TaskTagNames()
	if err != nil {
		this.showMsg("读取标签列表失败：" + err.Error())
	}
	this.Data["tagNames"] = tagNames
	this.Data["groups"] = groups
	this.Data["pageTitle"] = "添加任务"
	this.display()
}

// 编辑任务
func (this *TaskController) Edit() {
	id, _ := this.GetInt("id")

	task, err := models.TaskGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	existingTags, err := models.TaskTagsByTaskIDs([]int{task.Id})
	if err != nil {
		this.showMsg("读取任务标签失败：" + err.Error())
	}
	if this.isPost() {
		before := taskSnapshot(task)
		task.TaskName = strings.TrimSpace(this.GetString("task_name"))
		task.Description = strings.TrimSpace(this.GetString("description"))
		tags, err := models.ParseTaskTags(this.GetString("tags"))
		if err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		task.GroupId, _ = this.GetInt("group_id")
		task.Concurrent, _ = this.GetInt("concurrent")
		task.CronSpec = strings.TrimSpace(this.GetString("cron_spec"))
		task.Command = strings.TrimSpace(this.GetString("command"))
		task.Notify, _ = this.GetInt("notify")
		task.Timeout, _ = this.GetInt("timeout")

		notifyEmail := strings.TrimSpace(this.GetString("notify_email"))
		if notifyEmail != "" {
			tmp := strings.Split(notifyEmail, "\n")
			emailList := make([]string, 0, len(tmp))
			for _, v := range tmp {
				v = strings.TrimSpace(v)
				if !libs.IsEmail([]byte(v)) {
					this.ajaxMsg("无效的Email地址："+v, MSG_ERR)
				} else {
					emailList = append(emailList, v)
				}
			}
			task.NotifyEmail = strings.Join(emailList, "\n")
		}

		if task.TaskName == "" || task.CronSpec == "" || task.Command == "" {
			this.ajaxMsg("请填写完整信息", MSG_ERR)
		}
		if _, err := libcron.Parse(task.CronSpec); err != nil {
			this.ajaxMsg("cron表达式无效", MSG_ERR)
		}
		if err := models.TaskSaveWithTags(task, tags); err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		this.audit("task_update", "task", task.Id, models.OP_RESULT_OK, map[string]interface{}{
			"before":      before,
			"after":       taskSnapshot(task),
			"tags_before": existingTags[task.Id],
			"tags_after":  tags,
		})

		this.ajaxMsg("", MSG_OK)
	}

	// 分组列表
	groups, _ := models.TaskGroupGetList(1, 100)
	tagNames, err := models.TaskTagNames()
	if err != nil {
		this.showMsg("读取标签列表失败：" + err.Error())
	}
	this.Data["tagNames"] = tagNames
	this.Data["groups"] = groups
	this.Data["task"] = task
	this.Data["tagsText"] = strings.Join(existingTags[task.Id], ", ")
	this.Data["pageTitle"] = "编辑任务"
	this.display()
}

// 任务执行日志列表
func (this *TaskController) Logs() {
	taskId, _ := this.GetInt("id")
	page, _ := this.GetInt("page")
	if page < 1 {
		page = 1
	}

	task, err := models.TaskGetById(taskId)
	if err != nil {
		this.showMsg(err.Error())
	}

	result, count := models.TaskLogGetList(page, this.pageSize, "task_id", task.Id)

	list := make([]map[string]interface{}, len(result))
	for k, v := range result {
		row := make(map[string]interface{})
		row["id"] = v.Id
		row["start_time"] = beego.Date(time.Unix(v.CreateTime, 0), "Y-m-d H:i:s")
		row["process_time"] = float64(v.ProcessTime) / 1000
		row["ouput_size"] = libs.SizeFormat(float64(len(v.Output)))
		row["status"] = v.Status
		list[k] = row
	}

	this.Data["pageTitle"] = "任务执行日志"
	this.Data["list"] = list
	this.Data["task"] = task
	this.Data["pageBar"] = libs.NewPager(page, int(count), this.pageSize, beego.URLFor("TaskController.Logs", "id", taskId), true).ToString()
	this.display()
}

// 查看日志详情
func (this *TaskController) ViewLog() {
	id, _ := this.GetInt("id")

	taskLog, err := models.TaskLogGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	task, err := models.TaskGetById(taskLog.TaskId)
	if err != nil {
		this.showMsg(err.Error())
	}

	data := make(map[string]interface{})
	data["id"] = taskLog.Id
	data["output"] = taskLog.Output
	data["error"] = taskLog.Error
	data["start_time"] = beego.Date(time.Unix(taskLog.CreateTime, 0), "Y-m-d H:i:s")
	data["process_time"] = float64(taskLog.ProcessTime) / 1000
	data["ouput_size"] = libs.SizeFormat(float64(len(taskLog.Output)))
	data["status"] = taskLog.Status

	this.Data["task"] = task
	this.Data["data"] = data
	this.Data["pageTitle"] = "查看日志"
	this.display()
}

// 批量操作日志
func (this *TaskController) LogBatch() {
	action := this.GetString("action")
	ids := this.GetStrings("ids")
	if len(ids) < 1 {
		this.ajaxMsg("请选择要操作的项目", MSG_ERR)
	}
	for _, v := range ids {
		id, _ := strconv.Atoi(v)
		if id < 1 {
			continue
		}
		switch action {
		case "delete":
			if taskLog, err := models.TaskLogGetById(id); err == nil {
				models.TaskLogDelById(id)
				this.audit("task_log_delete", "task_log", id, models.OP_RESULT_OK, taskLogSnapshot(taskLog))
			}
		}
	}

	this.ajaxMsg("", MSG_OK)
}

// 批量操作
func (this *TaskController) Batch() {
	action := this.GetString("action")
	ids := this.GetStrings("ids")
	if len(ids) < 1 {
		this.ajaxMsg("请选择要操作的项目", MSG_ERR)
	}

	for _, v := range ids {
		id, _ := strconv.Atoi(v)
		if id < 1 {
			continue
		}
		switch action {
		case "active":
			if task, err := models.TaskGetById(id); err == nil {
				job, err := jobs.NewJobFromTask(task)
				if err == nil {
					jobs.AddJob(task.CronSpec, job)
					task.Status = 1
					task.Update()
					this.audit("task_batch_active", "task", task.Id, models.OP_RESULT_OK, taskSnapshot(task))
				}
			}
		case "pause":
			jobs.RemoveJob(id)
			if task, err := models.TaskGetById(id); err == nil {
				task.Status = 0
				task.Update()
				this.audit("task_batch_pause", "task", task.Id, models.OP_RESULT_OK, taskSnapshot(task))
			}
		case "delete":
			if task, err := models.TaskGetById(id); err == nil {
				before := taskSnapshot(task)
				models.TaskDel(id)
				models.TaskLogDelByTaskId(id)
				jobs.RemoveJob(id)
				this.audit("task_batch_delete", "task", id, models.OP_RESULT_OK, before)
			}
		}
	}

	this.ajaxMsg("", MSG_OK)
}

// 启动任务
func (this *TaskController) Start() {
	id, _ := this.GetInt("id")

	task, err := models.TaskGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	job, err := jobs.NewJobFromTask(task)
	if err != nil {
		this.showMsg(err.Error())
	}

	if jobs.AddJob(task.CronSpec, job) {
		task.Status = 1
		task.Update()
		this.audit("task_start", "task", task.Id, models.OP_RESULT_OK, taskSnapshot(task))
	}

	refer := this.Ctx.Request.Referer()
	if refer == "" {
		refer = beego.URLFor("TaskController.List")
	}
	this.redirect(refer)
}

// 暂停任务
func (this *TaskController) Pause() {
	id, _ := this.GetInt("id")

	task, err := models.TaskGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	jobs.RemoveJob(id)
	task.Status = 0
	task.Update()
	this.audit("task_pause", "task", task.Id, models.OP_RESULT_OK, taskSnapshot(task))

	refer := this.Ctx.Request.Referer()
	if refer == "" {
		refer = beego.URLFor("TaskController.List")
	}
	this.redirect(refer)
}

// 立即执行
func (this *TaskController) Run() {
	id, _ := this.GetInt("id")

	task, err := models.TaskGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	job, err := jobs.NewJobFromTask(task)
	if err != nil {
		this.showMsg(err.Error())
	}

	job.Run()
	this.audit("task_manual_run", "task", task.Id, models.OP_RESULT_OK, map[string]interface{}{
		"task":        taskSnapshot(task),
		"task_log_id": job.GetLogId(),
	})

	this.redirect(beego.URLFor("TaskController.ViewLog", "id", job.GetLogId()))
}

func taskSnapshot(task *models.Task) map[string]interface{} {
	return map[string]interface{}{
		"id":            task.Id,
		"user_id":       task.UserId,
		"group_id":      task.GroupId,
		"task_name":     task.TaskName,
		"task_type":     task.TaskType,
		"description":   task.Description,
		"cron_spec":     task.CronSpec,
		"concurrent":    task.Concurrent,
		"command":       task.Command,
		"status":        task.Status,
		"notify":        task.Notify,
		"notify_email":  task.NotifyEmail,
		"timeout":       task.Timeout,
		"execute_times": task.ExecuteTimes,
		"prev_time":     task.PrevTime,
		"create_time":   task.CreateTime,
	}
}

func taskLogSnapshot(log *models.TaskLog) map[string]interface{} {
	return map[string]interface{}{
		"id":           log.Id,
		"task_id":      log.TaskId,
		"status":       log.Status,
		"process_time": log.ProcessTime,
		"create_time":  log.CreateTime,
	}
}
