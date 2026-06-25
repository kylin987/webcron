package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/utils"
	"github.com/lisijie/webcron/app/libs"
	"github.com/lisijie/webcron/app/models"
	"strings"
	"time"
)

type UserController struct {
	BaseController
}

func (this *UserController) List() {
	this.requireAdmin()
	page, _ := this.GetInt("page")
	if page < 1 {
		page = 1
	}

	result, count := models.UserGetList(page, this.pageSize)
	list := make([]map[string]interface{}, len(result))
	for k, user := range result {
		row := userSnapshot(user)
		if user.LastLogin > 0 {
			row["last_login_text"] = beego.Date(time.Unix(user.LastLogin, 0), "Y-m-d H:i:s")
		} else {
			row["last_login_text"] = "-"
		}
		list[k] = row
	}

	this.Data["pageTitle"] = "账号列表"
	this.Data["list"] = list
	this.Data["pageBar"] = libs.NewPager(page, int(count), this.pageSize, beego.URLFor("UserController.List"), true).ToString()
	this.display()
}

func (this *UserController) Add() {
	this.requireAdmin()
	if this.isPost() {
		userName := strings.TrimSpace(this.GetString("user_name"))

		if userName == "" {
			this.ajaxMsg("请填写用户名", MSG_ERR)
		}
		if _, err := models.UserGetByName(userName); err == nil {
			this.ajaxMsg("用户名已存在", MSG_ERR)
		}

		// 密码由服务端随机生成，不在表单中填写，仅在创建成功时一次性返回
		password := libs.GeneratePassword(16)

		user := &models.User{
			UserName: userName,
			Salt:     string(utils.RandomCreateBytes(10)),
			Status:   0,
		}
		user.Password = libs.Md5([]byte(password + user.Salt))
		id, err := models.UserAdd(user)
		if err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}
		user.Id = int(id)
		this.audit("user_create", "user", user.Id, models.OP_RESULT_OK, userSnapshot(user))

		// 仅本次返回明文密码，供管理员一次性转交；后续列表/编辑均不再展示
		this.ajaxMsg(map[string]interface{}{
			"user_name": userName,
			"password":  password,
		}, MSG_OK)
	}

	this.Data["pageTitle"] = "添加账号"
	this.display()
}

func (this *UserController) Edit() {
	this.requireAdmin()
	id, _ := this.GetInt("id")
	user, err := models.UserGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}

	if this.isPost() {
		before := userSnapshot(user)
		status, _ := this.GetInt("status")
		password := strings.TrimSpace(this.GetString("password"))

		if user.Id == this.userId && status == -1 {
			this.ajaxMsg("不能禁用当前登录账号", MSG_ERR)
		}
		user.Status = status
		if password != "" {
			if len(password) < 8 {
				this.ajaxMsg("密码长度必须至少8位", MSG_ERR)
			}
			user.Salt = string(utils.RandomCreateBytes(10))
			user.Password = libs.Md5([]byte(password + user.Salt))
		}
		if err := user.Update(); err != nil {
			this.ajaxMsg(err.Error(), MSG_ERR)
		}

		action := "user_update"
		if password != "" {
			action = "user_reset_password"
		} else if before["status"] != user.Status {
			if user.Status == -1 {
				action = "user_disable"
			} else {
				action = "user_enable"
			}
		}
		this.audit(action, "user", user.Id, models.OP_RESULT_OK, map[string]interface{}{
			"before": before,
			"after":  userSnapshot(user),
		})
		this.ajaxMsg("", MSG_OK)
	}

	this.Data["pageTitle"] = "编辑账号"
	this.Data["user"] = user
	this.display()
}

func (this *UserController) ResetMfa() {
	this.requireAdmin()
	id, _ := this.GetInt("id")
	user, err := models.UserGetById(id)
	if err != nil {
		this.showMsg(err.Error())
	}
	before := userSnapshot(user)
	user.MfaSecret = ""
	user.MfaEnabled = 0
	user.MfaConfirmedAt = 0
	if err := user.Update(); err != nil {
		this.showMsg(err.Error())
	}
	this.audit("user_reset_mfa", "user", user.Id, models.OP_RESULT_OK, map[string]interface{}{
		"before": before,
		"after":  userSnapshot(user),
	})
	this.redirect(beego.URLFor("UserController.List"))
}

func userSnapshot(user *models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":               user.Id,
		"user_name":        user.UserName,
		"status":           user.Status,
		"mfa_enabled":      user.MfaEnabled,
		"mfa_confirmed_at": user.MfaConfirmedAt,
		"last_login":       user.LastLogin,
		"last_ip":          user.LastIp,
		"create_time":      user.CreateTime,
	}
}
