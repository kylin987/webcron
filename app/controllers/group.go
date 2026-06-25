package controllers

import (
	"github.com/astaxie/beego"
	"github.com/lisijie/webcron/app/libs"
	"github.com/lisijie/webcron/app/models"
	"strconv"
	"strings"
)

type GroupController struct {
	BaseController
}

func (this *GroupController) List() {
	page, _ := this.GetInt("page")
	if page < 1 {
		page = 1
	}

	list, count := models.TaskGroupGetList(page, this.pageSize)

	this.Data["pageTitle"] = "分组列表"
	this.Data["list"] = list
	this.Data["pageBar"] = libs.NewPager(page, int(count), this.pageSize, beego.URLFor("GroupController.List"), true).ToString()
	this.display()
}

func (this *GroupController) Add() {
	if this.isPost() {
		group := new(models.TaskGroup)
		group.GroupName = strings.TrimSpace(this.GetString("group_name"))
		group.UserId = this.userId
		group.Description = strings.TrimSpace(this.GetString("description"))

		id, err := models.TaskGroupAdd(group)
		if err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		group.Id = int(id)
		this.audit("group_create", "group", group.Id, models.OP_RESULT_OK, groupSnapshot(group))
		this.ajaxMsg("", MSG_OK)
	}

	this.Data["pageTitle"] = "添加分组"
	this.display()
}

func (this *GroupController) Edit() {
	id, _ := this.GetInt("id")

	group, err := models.TaskGroupGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	if this.isPost() {
		before := groupSnapshot(group)
		group.GroupName = strings.TrimSpace(this.GetString("group_name"))
		group.Description = strings.TrimSpace(this.GetString("description"))
		err := group.Update()
		if err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		this.audit("group_update", "group", group.Id, models.OP_RESULT_OK, map[string]interface{}{
			"before": before,
			"after":  groupSnapshot(group),
		})
		this.ajaxMsg("", MSG_OK)
	}

	this.Data["pageTitle"] = "编辑分组"
	this.Data["group"] = group
	this.display()
}

func (this *GroupController) Batch() {
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
			if group, err := models.TaskGroupGetById(id); err == nil {
				before := groupSnapshot(group)
				models.TaskGroupDelById(id)
				models.TaskResetGroupId(id)
				this.audit("group_batch_delete", "group", id, models.OP_RESULT_OK, before)
			}
		}
	}

	this.ajaxMsg("", MSG_OK)
}

func groupSnapshot(group *models.TaskGroup) map[string]interface{} {
	return map[string]interface{}{
		"id":          group.Id,
		"user_id":     group.UserId,
		"group_name":  group.GroupName,
		"description": group.Description,
		"create_time": group.CreateTime,
	}
}
