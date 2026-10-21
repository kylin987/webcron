package jobs

import (
	"github.com/astaxie/beego"
	"github.com/lisijie/cron"
	"sync"
)

var (
	mainCron *cron.Cron
	workPool chan bool
	lock     sync.Mutex
)

func init() {
	if size, _ := beego.AppConfig.Int("jobs.pool"); size > 0 {
		workPool = make(chan bool, size)
	}
	mainCron = cron.New()
	// 本地模式下不启动调度器，避免定时任务被自动触发；
	// 手动执行（Job.Run）不经过调度器，仍然可用。
	if !IsLocalMode() {
		mainCron.Start()
	}
}

// IsLocalMode 本地模式下不自动执行任何定时任务，仅保留手动运行与页面操作。
// 通过配置 jobs.local = true 开启，便于本地开发调试时连线上库而不触发真实任务。
func IsLocalMode() bool {
	v, _ := beego.AppConfig.Bool("jobs.local")
	return v
}

func AddJob(spec string, job *Job) bool {
	if IsLocalMode() {
		beego.Info("本地模式已开启，跳过任务调度注册: ", job.id)
		return false
	}

	lock.Lock()
	defer lock.Unlock()

	if GetEntryById(job.id) != nil {
		return false
	}
	err := mainCron.AddJob(spec, job)
	if err != nil {
		beego.Error("AddJob: ", err.Error())
		return false
	}
	return true
}

func RemoveJob(id int) {
	mainCron.RemoveJob(func(e *cron.Entry) bool {
		if v, ok := e.Job.(*Job); ok {
			if v.id == id {
				return true
			}
		}
		return false
	})
}

// AddSystemJob 注册系统级定时任务（不依赖 t_task 表，如日志清理）。
// 本地模式下跳过，避免本地连线上库触发清理。
func AddSystemJob(spec string, cmd func()) error {
	if IsLocalMode() {
		beego.Info("本地模式已开启，跳过系统任务注册")
		return nil
	}
	lock.Lock()
	defer lock.Unlock()
	return mainCron.AddFunc(spec, cmd)
}

func GetEntryById(id int) *cron.Entry {
	entries := mainCron.Entries()
	for _, e := range entries {
		if v, ok := e.Job.(*Job); ok {
			if v.id == id {
				return e
			}
		}
	}
	return nil
}

func GetEntries(size int) []*cron.Entry {
	ret := make([]*cron.Entry, 0)
	for _, e := range mainCron.Entries() {
		if _, ok := e.Job.(*Job); !ok {
			continue
		}
		ret = append(ret, e)
		if size > 0 && len(ret) >= size {
			break
		}
	}
	return ret
}
