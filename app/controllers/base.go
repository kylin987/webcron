package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego"
	"github.com/lisijie/webcron/app/libs"
	"github.com/lisijie/webcron/app/models"
	"net"
	"strconv"
	"strings"
)

const (
	MSG_OK  = 0
	MSG_ERR = -1
)

type BaseController struct {
	beego.Controller
	controllerName string
	actionName     string
	user           *models.User
	userId         int
	userName       string
	pageSize       int
}

func (this *BaseController) Prepare() {
	this.pageSize = 20
	controllerName, actionName := this.GetControllerAndAction()
	this.controllerName = strings.ToLower(controllerName[0 : len(controllerName)-10])
	this.actionName = strings.ToLower(actionName)
	this.auth()

	this.Data["version"] = beego.AppConfig.String("version")
	this.Data["siteName"] = beego.AppConfig.String("site.name")
	this.Data["curRoute"] = this.controllerName + "." + this.actionName
	this.Data["curController"] = this.controllerName
	this.Data["curAction"] = this.actionName
	this.Data["loginUserId"] = this.userId
	this.Data["loginUserName"] = this.userName
	this.Data["isAdmin"] = this.isAdmin()
}

// isAdmin 仅 admin（id=1）拥有账号管理与操作审计权限。
func (this *BaseController) isAdmin() bool {
	return this.userId == 1
}

// requireAdmin 非管理员访问受保护资源时拒绝并跳转。
func (this *BaseController) requireAdmin() {
	if !this.isAdmin() {
		flash := beego.NewFlash()
		flash.Error("无权限访问")
		flash.Store(&this.Controller)
		this.redirect(beego.URLFor("TaskController.List"))
	}
}

// 登录状态验证
func (this *BaseController) auth() {
	arr := strings.Split(this.Ctx.GetCookie("auth"), "|")
	if len(arr) == 2 {
		idstr, password := arr[0], arr[1]
		userId, _ := strconv.Atoi(idstr)
		if userId > 0 {
			user, err := models.UserGetById(userId)
			if err == nil && user.Status == 0 && password == this.authKey(user) {
				if user.MfaEnabled != 1 {
					this.SetSession("pending_mfa_user_id", user.Id)
					this.Ctx.SetCookie("auth", "")
					if !this.isAuthFreeAction() && !(this.controllerName == "main" && this.actionName == "mfasetup") {
						this.redirect(beego.URLFor("MainController.MfaSetup"))
					}
					return
				}
				this.setLoginUser(user)
			}
		}
	}

	if this.userId == 0 && !this.isAuthFreeAction() {
		this.redirect(beego.URLFor("MainController.Login"))
	}
}

func (this *BaseController) isAuthFreeAction() bool {
	if this.controllerName != "main" {
		return false
	}
	switch this.actionName {
	case "login", "logout", "mfasetup", "mfaverify":
		return true
	default:
		return false
	}
}

func (this *BaseController) setLoginUser(user *models.User) {
	this.userId = user.Id
	this.userName = user.UserName
	this.user = user
}

func (this *BaseController) authKey(user *models.User) string {
	parts := []string{
		this.getClientIp(),
		user.Password,
		user.Salt,
		strconv.Itoa(user.MfaEnabled),
		strconv.FormatInt(user.MfaConfirmedAt, 10),
	}
	return libs.Md5([]byte(strings.Join(parts, "|")))
}

func (this *BaseController) setAuthCookie(user *models.User, remember bool) {
	cookie := strconv.Itoa(user.Id) + "|" + this.authKey(user)
	if remember {
		this.Ctx.SetCookie("auth", cookie, 7*86400)
	} else {
		this.Ctx.SetCookie("auth", cookie)
	}
}

func (this *BaseController) audit(action, targetType string, targetId int, result int, detail interface{}) {
	this.auditWithUser(this.userId, this.userName, action, targetType, targetId, result, detail)
}

func (this *BaseController) auditWithUser(userId int, userName, action, targetType string, targetId int, result int, detail interface{}) {
	detailText := ""
	if detail != nil {
		buf, err := json.Marshal(detail)
		if err == nil {
			detailText = string(buf)
		}
	}

	userAgent := this.Ctx.Input.UserAgent()
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	requestPath := ""
	if this.Ctx.Request != nil && this.Ctx.Request.URL != nil {
		requestPath = this.Ctx.Request.URL.RequestURI()
		if len(requestPath) > 255 {
			requestPath = requestPath[:255]
		}
	}

	_, err := models.OperationLogAdd(&models.OperationLog{
		UserId:        userId,
		UserName:      userName,
		Action:        action,
		TargetType:    targetType,
		TargetId:      targetId,
		Result:        result,
		Ip:            this.getClientIp(),
		UserAgent:     userAgent,
		RequestMethod: this.Ctx.Request.Method,
		RequestPath:   requestPath,
		Detail:        detailText,
	})
	if err != nil {
		beego.Warn("operation log write failed:", err)
	}
}

// 渲染模版
func (this *BaseController) display(tpl ...string) {
	var tplname string
	if len(tpl) > 0 {
		tplname = tpl[0] + ".html"
	} else {
		tplname = this.controllerName + "/" + this.actionName + ".html"
	}
	this.Layout = "layout/layout.html"
	this.TplName = tplname
}

// 重定向
func (this *BaseController) redirect(url string) {
	this.Redirect(url, 302)
	this.StopRun()
}

// 是否POST提交
func (this *BaseController) isPost() bool {
	return this.Ctx.Request.Method == "POST"
}

// 显示错误信息
func (this *BaseController) showMsg(args ...string) {
	this.Data["message"] = args[0]
	redirect := this.Ctx.Request.Referer()
	if len(args) > 1 {
		redirect = args[1]
	}

	this.Data["redirect"] = redirect
	this.Data["pageTitle"] = "系统提示"
	this.display("error/message")
	this.Render()
	this.StopRun()
}

// 输出json
func (this *BaseController) jsonResult(out interface{}) {
	this.Data["json"] = out
	this.ServeJSON()
	this.StopRun()
}

func (this *BaseController) ajaxMsg(msg interface{}, msgno int) {
	out := make(map[string]interface{})
	out["status"] = msgno
	out["msg"] = msg

	this.jsonResult(out)
}

// 获取用户IP地址
func (this *BaseController) getClientIp() string {
	if ip := strings.TrimSpace(this.Ctx.Input.Header("X-Real-IP")); ip != "" {
		return ip
	}
	if forwardedFor := strings.TrimSpace(this.Ctx.Input.Header("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(this.Ctx.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return this.Ctx.Request.RemoteAddr
}
