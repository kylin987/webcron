package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/utils"
	"github.com/lisijie/webcron/app/jobs"
	"github.com/lisijie/webcron/app/libs"
	"github.com/lisijie/webcron/app/models"
	"html/template"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type MainController struct {
	BaseController
}

// 首页
func (this *MainController) Index() {
	this.Data["pageTitle"] = "系统概况"

	// 即将执行的任务
	entries := jobs.GetEntries(30)
	jobList := make([]map[string]interface{}, len(entries))
	for k, v := range entries {
		row := make(map[string]interface{})
		job := v.Job.(*jobs.Job)
		row["task_id"] = job.GetId()
		row["task_name"] = job.GetName()
		row["next_time"] = beego.Date(v.Next, "Y-m-d H:i:s")
		jobList[k] = row
	}

	// 最近执行的日志
	logs, _ := models.TaskLogGetList(1, 20)
	recentLogs := make([]map[string]interface{}, len(logs))
	for k, v := range logs {
		task, err := models.TaskGetById(v.TaskId)
		taskName := ""
		if err == nil {
			taskName = task.TaskName
		}
		row := make(map[string]interface{})
		row["task_name"] = taskName
		row["id"] = v.Id
		row["start_time"] = beego.Date(time.Unix(v.CreateTime, 0), "Y-m-d H:i:s")
		row["process_time"] = float64(v.ProcessTime) / 1000
		row["ouput_size"] = libs.SizeFormat(float64(len(v.Output)))
		row["output"] = beego.Substr(v.Output, 0, 100)
		row["status"] = v.Status
		recentLogs[k] = row
	}

	// 最近执行失败的日志
	logs, _ = models.TaskLogGetList(1, 20, "status__lt", 0)
	errLogs := make([]map[string]interface{}, len(logs))
	for k, v := range logs {
		task, err := models.TaskGetById(v.TaskId)
		taskName := ""
		if err == nil {
			taskName = task.TaskName
		}
		row := make(map[string]interface{})
		row["task_name"] = taskName
		row["id"] = v.Id
		row["start_time"] = beego.Date(time.Unix(v.CreateTime, 0), "Y-m-d H:i:s")
		row["process_time"] = float64(v.ProcessTime) / 1000
		row["ouput_size"] = libs.SizeFormat(float64(len(v.Output)))
		row["error"] = beego.Substr(v.Error, 0, 100)
		row["status"] = v.Status
		errLogs[k] = row
	}

	this.Data["recentLogs"] = recentLogs
	this.Data["errLogs"] = errLogs
	this.Data["jobs"] = jobList
	this.Data["cpuNum"] = runtime.NumCPU()
	this.display()
}

// 个人信息
func (this *MainController) Profile() {
	beego.ReadFromRequest(&this.Controller)
	user, _ := models.UserGetById(this.userId)

	if this.isPost() {
		flash := beego.NewFlash()
		before := userSnapshot(user)
		email := strings.TrimSpace(this.GetString("email"))
		password1 := this.GetString("password1")
		password2 := this.GetString("password2")
		passwordChanged := false
		if password1 != "" {
			if len(password1) < 8 {
				flash.Error("密码长度必须至少8位")
				flash.Store(&this.Controller)
				this.redirect(beego.URLFor(".Profile"))
			} else if password2 != password1 {
				flash.Error("两次输入的密码不一致")
				flash.Store(&this.Controller)
				this.redirect(beego.URLFor(".Profile"))
			} else {
				user.Salt = string(utils.RandomCreateBytes(10))
				user.Password = libs.Md5([]byte(password1 + user.Salt))
				passwordChanged = true
			}
		}
		user.Email = email
		user.Update()
		after := userSnapshot(user)
		if passwordChanged {
			this.audit("user_reset_password", "user", user.Id, models.OP_RESULT_OK, map[string]interface{}{
				"before": before,
				"after":  after,
				"scope":  "profile",
			})
		}
		this.audit("user_update", "user", user.Id, models.OP_RESULT_OK, map[string]interface{}{
			"before": before,
			"after":  after,
			"scope":  "profile",
		})
		flash.Success("修改成功！")
		flash.Store(&this.Controller)
		this.redirect(beego.URLFor(".Profile"))
	}

	this.Data["pageTitle"] = "个人信息"
	this.Data["user"] = user
	this.display()
}

// 登录
func (this *MainController) Login() {
	if this.userId > 0 {
		this.redirect("/")
	}
	beego.ReadFromRequest(&this.Controller)
	if this.isPost() {
		flash := beego.NewFlash()

		username := strings.TrimSpace(this.GetString("username"))
		password := strings.TrimSpace(this.GetString("password"))
		remember := this.GetString("remember") != ""
		if username != "" && password != "" {
			user, err := models.UserGetByName(username)
			errorMsg := ""
			if err != nil || user.Password != libs.Md5([]byte(password+user.Salt)) {
				errorMsg = "帐号或密码错误"
				this.auditWithUser(0, "", "login_failed", "user", 0, models.OP_RESULT_ERR, map[string]interface{}{
					"user_name": username,
					"reason":    "invalid_password",
				})
			} else if user.Status == -1 {
				errorMsg = "该帐号已禁用"
				this.auditWithUser(user.Id, user.UserName, "login_failed", "user", user.Id, models.OP_RESULT_ERR, map[string]interface{}{
					"user_name": user.UserName,
					"reason":    "disabled",
				})
			} else {
				this.setPendingMfa(user.Id, remember)
				if user.MfaEnabled == 1 {
					this.redirect(beego.URLFor("MainController.MfaVerify"))
				}
				this.redirect(beego.URLFor("MainController.MfaSetup"))
			}
			flash.Error("%s", errorMsg)
			flash.Store(&this.Controller)
			this.redirect(beego.URLFor("MainController.Login"))
		}
	}

	this.TplName = "main/login.html"
}

func (this *MainController) MfaSetup() {
	user, ok := this.pendingMfaUser()
	if !ok {
		this.redirect(beego.URLFor("MainController.Login"))
	}
	if user.MfaEnabled == 1 {
		this.redirect(beego.URLFor("MainController.MfaVerify"))
	}

	secret := this.pendingMfaSecret()
	if secret == "" {
		var err error
		secret, err = libs.NewTOTPSecret()
		if err != nil {
			this.showMsg("生成MFA密钥失败")
		}
		this.SetSession("pending_mfa_secret", secret)
	}

	if this.isPost() {
		code := strings.TrimSpace(this.GetString("code"))
		if libs.VerifyTOTP(secret, code, time.Now()) {
			user.MfaSecret = secret
			user.MfaEnabled = 1
			user.MfaConfirmedAt = time.Now().Unix()
			user.Update()
			this.auditWithUser(user.Id, user.UserName, "mfa_setup_success", "user", user.Id, models.OP_RESULT_OK, map[string]interface{}{
				"user_name": user.UserName,
			})
			this.finishMfaLogin(user)
		}
		this.auditWithUser(user.Id, user.UserName, "mfa_setup_failed", "user", user.Id, models.OP_RESULT_ERR, map[string]interface{}{
			"user_name": user.UserName,
		})
		flash := beego.NewFlash()
		flash.Error("动态验证码错误")
		flash.Store(&this.Controller)
		this.redirect(beego.URLFor("MainController.MfaSetup"))
	}

	beego.ReadFromRequest(&this.Controller)
	this.Data["pageTitle"] = "绑定MFA"
	this.Data["user"] = user
	qrCode, err := libs.TOTPQRCodeBase64(libs.TOTPURI(beego.AppConfig.String("site.name"), user.UserName, secret))
	if err != nil {
		this.showMsg("生成二维码失败")
	}
	this.Data["qrCode"] = template.URL(qrCode)
	this.TplName = "main/mfa_setup.html"
}

func (this *MainController) MfaVerify() {
	user, ok := this.pendingMfaUser()
	if !ok {
		this.redirect(beego.URLFor("MainController.Login"))
	}
	if user.MfaEnabled != 1 {
		this.redirect(beego.URLFor("MainController.MfaSetup"))
	}

	if this.isPost() {
		code := strings.TrimSpace(this.GetString("code"))
		if libs.VerifyTOTP(user.MfaSecret, code, time.Now()) {
			this.finishMfaLogin(user)
		}
		this.auditWithUser(user.Id, user.UserName, "mfa_verify_failed", "user", user.Id, models.OP_RESULT_ERR, map[string]interface{}{
			"user_name": user.UserName,
		})
		flash := beego.NewFlash()
		flash.Error("动态验证码错误")
		flash.Store(&this.Controller)
		this.redirect(beego.URLFor("MainController.MfaVerify"))
	}

	beego.ReadFromRequest(&this.Controller)
	this.Data["pageTitle"] = "MFA验证"
	this.Data["user"] = user
	this.TplName = "main/mfa_verify.html"
}

// 退出登录
func (this *MainController) Logout() {
	if this.userId > 0 {
		this.audit("logout", "user", this.userId, models.OP_RESULT_OK, map[string]interface{}{
			"user_name": this.userName,
		})
	}
	this.Ctx.SetCookie("auth", "")
	this.redirect(beego.URLFor("MainController.Login"))
}

// 获取系统时间
func (this *MainController) GetTime() {
	out := make(map[string]interface{})
	out["time"] = time.Now().UnixNano() / int64(time.Millisecond)
	this.jsonResult(out)
}

func (this *MainController) setPendingMfa(userId int, remember bool) {
	this.SetSession("pending_mfa_user_id", userId)
	if remember {
		this.SetSession("pending_mfa_remember", 1)
	} else {
		this.SetSession("pending_mfa_remember", 0)
	}
	this.DelSession("pending_mfa_secret")
}

func (this *MainController) pendingMfaUser() (*models.User, bool) {
	v := this.GetSession("pending_mfa_user_id")
	userId := 0
	switch val := v.(type) {
	case int:
		userId = val
	case int64:
		userId = int(val)
	case string:
		userId, _ = strconv.Atoi(val)
	}
	if userId < 1 {
		return nil, false
	}
	user, err := models.UserGetById(userId)
	if err != nil || user.Status == -1 {
		return nil, false
	}
	return user, true
}

func (this *MainController) pendingMfaSecret() string {
	if secret, ok := this.GetSession("pending_mfa_secret").(string); ok {
		return secret
	}
	return ""
}

func (this *MainController) finishMfaLogin(user *models.User) {
	user.LastIp = this.getClientIp()
	user.LastLogin = time.Now().Unix()
	models.UserUpdate(user)

	remember := false
	if value, ok := this.GetSession("pending_mfa_remember").(int); ok && value == 1 {
		remember = true
	}
	this.setLoginUser(user)
	this.setAuthCookie(user, remember)
	this.DelSession("pending_mfa_user_id")
	this.DelSession("pending_mfa_remember")
	this.DelSession("pending_mfa_secret")
	this.auditWithUser(user.Id, user.UserName, "login_success", "user", user.Id, models.OP_RESULT_OK, map[string]interface{}{
		"user_name": user.UserName,
	})
	this.redirect(beego.URLFor("TaskController.List"))
}
