package controllers

import (
	"bytes"
	"encoding/json"
	"github.com/astaxie/beego"
	"github.com/lisijie/webcron/app/libs"
	"github.com/lisijie/webcron/app/models"
	"strconv"
	"strings"
	"time"
)

type OperationLogController struct {
	BaseController
}

func (this *OperationLogController) List() {
	this.requireAdmin()
	page, _ := this.GetInt("page")
	if page < 1 {
		page = 1
	}

	userId, _ := this.GetInt("user_id")
	action := strings.TrimSpace(this.GetString("action"))
	targetType := strings.TrimSpace(this.GetString("target_type"))
	targetId, _ := this.GetInt("target_id")
	result := strings.TrimSpace(this.GetString("result"))
	startTime := strings.TrimSpace(this.GetString("start_time"))
	endTime := strings.TrimSpace(this.GetString("end_time"))

	filters := make([]interface{}, 0)
	if userId > 0 {
		filters = append(filters, "user_id", userId)
	}
	if action != "" {
		filters = append(filters, "action", action)
	}
	if targetType != "" {
		filters = append(filters, "target_type", targetType)
	}
	if targetId > 0 {
		filters = append(filters, "target_id", targetId)
	}
	if result != "" {
		resultInt, _ := strconv.Atoi(result)
		filters = append(filters, "result", resultInt)
	}
	if ts := parseDateTime(startTime); ts > 0 {
		filters = append(filters, "create_time__gte", ts)
	}
	if ts := parseDateTime(endTime); ts > 0 {
		filters = append(filters, "create_time__lte", ts)
	}

	resultList, count := models.OperationLogGetList(page, this.pageSize, filters...)
	list := make([]map[string]interface{}, len(resultList))
	for k, log := range resultList {
		list[k] = operationLogRow(log)
	}

	users, _ := models.UserGetList(1, 1000)
	this.Data["pageTitle"] = "操作审计"
	this.Data["list"] = list
	this.Data["users"] = users
	this.Data["actions"] = auditActionLabels()
	this.Data["actionOptions"] = auditActionOptions()
	this.Data["user_id"] = userId
	this.Data["action"] = action
	this.Data["target_type"] = targetType
	this.Data["target_id"] = targetId
	this.Data["result"] = result
	this.Data["start_time"] = startTime
	this.Data["end_time"] = endTime
	this.Data["pageBar"] = libs.NewPager(page, int(count), this.pageSize, beego.URLFor("OperationLogController.List", "user_id", userId, "action", action, "target_type", targetType, "target_id", targetId, "result", result, "start_time", startTime, "end_time", endTime), true).ToString()
	this.display()
}

func (this *OperationLogController) View() {
	this.requireAdmin()
	id, _ := this.GetInt64("id")
	log, err := models.OperationLogGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	data := operationLogRow(log)
	data["detail_pretty"] = prettyJSON(log.Detail)
	this.Data["pageTitle"] = "审计详情"
	this.Data["data"] = data
	this.display()
}

func operationLogRow(log *models.OperationLog) map[string]interface{} {
	actionLabels := auditActionLabels()
	actionLabel := actionLabels[log.Action]
	if actionLabel == "" {
		actionLabel = log.Action
	}
	return map[string]interface{}{
		"id":             log.Id,
		"user_id":        log.UserId,
		"user_name":      log.UserName,
		"action":         log.Action,
		"action_label":   actionLabel,
		"target_type":    log.TargetType,
		"target_id":      log.TargetId,
		"result":         log.Result,
		"ip":             log.Ip,
		"user_agent":     log.UserAgent,
		"request_method": log.RequestMethod,
		"request_path":   log.RequestPath,
		"detail":         log.Detail,
		"create_time":    beego.Date(time.Unix(log.CreateTime, 0), "Y-m-d H:i:s"),
	}
}

func parseDateTime(value string) int64 {
	if value == "" {
		return 0
	}
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return t.Unix()
		}
	}
	return 0
}

func prettyJSON(value string) string {
	if value == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(value), "", "  "); err != nil {
		return value
	}
	return buf.String()
}

func auditActionLabels() map[string]string {
	return map[string]string{
		"login_success":       "登录成功",
		"login_failed":        "登录失败",
		"logout":              "退出登录",
		"mfa_setup_success":   "MFA绑定成功",
		"mfa_setup_failed":    "MFA绑定失败",
		"mfa_verify_failed":   "MFA验证失败",
		"user_create":         "创建账号",
		"user_update":         "更新账号",
		"user_enable":         "启用账号",
		"user_disable":        "禁用账号",
		"user_reset_password": "重置密码",
		"user_reset_mfa":      "重置MFA",
		"task_create":         "创建任务",
		"task_update":         "更新任务",
		"task_start":          "激活任务",
		"task_pause":          "暂停任务",
		"task_delete":         "删除任务",
		"task_batch_active":   "批量激活任务",
		"task_batch_pause":    "批量暂停任务",
		"task_batch_delete":   "批量删除任务",
		"task_manual_run":     "手动执行任务",
		"task_log_delete":     "删除任务日志",
		"group_create":        "创建分组",
		"group_update":        "更新分组",
		"group_delete":        "删除分组",
		"group_batch_delete":  "批量删除分组",
	}
}

func auditActionOptions() []map[string]string {
	actions := []string{
		"login_success",
		"login_failed",
		"logout",
		"mfa_setup_success",
		"mfa_setup_failed",
		"mfa_verify_failed",
		"user_create",
		"user_update",
		"user_enable",
		"user_disable",
		"user_reset_password",
		"user_reset_mfa",
		"task_create",
		"task_update",
		"task_start",
		"task_pause",
		"task_delete",
		"task_batch_active",
		"task_batch_pause",
		"task_batch_delete",
		"task_manual_run",
		"task_log_delete",
		"group_create",
		"group_update",
		"group_delete",
		"group_batch_delete",
	}
	labels := auditActionLabels()
	options := make([]map[string]string, len(actions))
	for i, action := range actions {
		options[i] = map[string]string{
			"key":   action,
			"label": labels[action],
		}
	}
	return options
}
