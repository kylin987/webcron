package jobs

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/lisijie/webcron/app/models"
)

const cleanupCronSpec = "0 0 5 * * *"

// RegisterCleanupJob 注册任务执行记录定时清理任务，每天凌晨 5 点执行。
// 保留天数由配置项 log.cleanup_days 控制，默认 30 天。
func RegisterCleanupJob() {
	days, _ := beego.AppConfig.Int("log.cleanup_days")
	if days <= 0 {
		days = 30
	}

	err := AddSystemJob(cleanupCronSpec, func() {
		before := time.Now().Unix() - int64(days)*86400
		n, err := models.TaskLogDelByTime(before)
		if err != nil {
			beego.Error("清理任务执行记录失败: ", err.Error())
			return
		}
		beego.Info("清理任务执行记录完成: 删除 ", n, " 条, 保留 ", days, " 天")
	})
	if err != nil {
		beego.Error("注册清理任务失败: ", err.Error())
	}
}
