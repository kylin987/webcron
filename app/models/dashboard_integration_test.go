package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

// Opt in with a disposable MySQL database. Every created table has a unique prefix.
func TestDashboardMySQL(t *testing.T) {
	dsn := os.Getenv("WEBCRON_TEST_DSN")
	if dsn == "" {
		t.Skip("WEBCRON_TEST_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 30; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("wc_test_%d_", time.Now().UnixNano())
	execSQL := func(statement string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(statement, args...); err != nil {
			t.Fatal(err)
		}
	}
	runSQL := func(source string) {
		t.Helper()
		for _, statement := range strings.Split(source, ";") {
			if strings.TrimSpace(statement) != "" {
				execSQL(statement)
			}
		}
	}
	t.Cleanup(func() {
		for _, name := range []string{"task_tag", "task_log", "task_group", "task", "user", "operation_log", "legacy_task", "legacy_task_log"} {
			db.Exec("DROP TABLE IF EXISTS `" + prefix + name + "`")
		}
	})
	source, err := os.ReadFile(filepath.Join("..", "..", "install.sql"))
	if err != nil {
		t.Fatal(err)
	}
	runSQL(strings.ReplaceAll(string(source), "`t_", "`"+prefix))
	beego.AppConfig.Set("db.prefix", prefix)
	if err := orm.RegisterDataBase("default", "mysql", dsn); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(Task), new(TaskTag), new(TaskLog))
	orm.BootStrap()
	task := &Task{TaskName: "订单同步", CronSpec: "0 * * * * *", Command: "echo test", GroupId: 3}
	if err := TaskSaveWithTags(task, []string{"生产"}); err != nil {
		t.Fatal(err)
	}
	stale, err := TaskGetById(task.Id)
	if err != nil {
		t.Fatal(err)
	}
	record := func(status int, output, errorText string) int64 {
		t.Helper()
		id, err := TaskLogAdd(&TaskLog{TaskId: task.Id, Status: status, Output: output, Error: errorText, CreateTime: time.Now().Unix(), ProcessTime: 50})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	record(TASK_ERROR, strings.Repeat("输", 10000), strings.Repeat("错", 400))
	state, err := TaskGetById(task.Id)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastStatus != TASK_ERROR || utf8.RuneCountInString(state.LastError) != 300 {
		t.Fatalf("bad snapshot: %#v", state)
	}
	summary, err := CurrentFailedTasks()
	if err != nil || summary.Count != 1 || len(summary.Tasks) != 1 {
		t.Fatalf("summary=%#v err=%v", summary, err)
	}
	stale.Status = 1
	if err := stale.Update(); err != nil {
		t.Fatal(err)
	}
	if err := TaskSaveWithTags(stale, []string{"生产", "同步"}); err != nil {
		t.Fatal(err)
	}
	state, err = TaskGetById(task.Id)
	if err != nil || state.LastStatus != TASK_ERROR {
		t.Fatalf("stale config overwrote result: %v", err)
	}
	_, count := TaskGetList(1, 20, "Tags__Name", "生产", "group_id", 3, "last_status__lt", 0)
	if count != 1 {
		t.Fatal("combined failure filter")
	}
	logs, err := DashboardLogs(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || utf8.RuneCountInString(logs[0].Output) != 100 || logs[0].OutputSize != 30000 || logs[0].TaskName != task.TaskName {
		t.Fatalf("bad summary rows %#v", logs)
	}
	record(TASK_TIMEOUT, "", "timeout")
	record(TASK_SUCCESS, "ok", "")
	summary, err = CurrentFailedTasks()
	if err != nil || summary.Count != 0 {
		t.Fatalf("recovery summary=%#v err=%v", summary, err)
	}
	for i := 0; i < 30; i++ {
		record([]int{TASK_SUCCESS, TASK_ERROR, TASK_TIMEOUT}[i%3], "o", "e")
	}
	logs, err = DashboardLogs(true)
	if err != nil || len(logs) != 20 {
		t.Fatalf("failed logs length %d, err %v", len(logs), err)
	}
	for i, l := range logs {
		if l.Status >= 0 || (i > 0 && logs[i-1].Id <= l.Id) {
			t.Fatal("failed log order/status")
		}
	}
	var logCount int
	db.QueryRow("SELECT COUNT(*) FROM `" + prefix + "task_log`").Scan(&logCount)
	// Failed log insertion must not leave either half of the transaction behind.
	_, err = TaskLogAdd(&TaskLog{TaskId: task.Id, Output: strings.Repeat("x", 1<<16), Error: strings.Repeat("x", 1<<16)})
	if err == nil {
		t.Fatal("expected oversized error failure")
	}
	var after int
	db.QueryRow("SELECT COUNT(*) FROM `" + prefix + "task_log`").Scan(&after)
	if after != logCount {
		t.Fatal("failed insert was committed")
	}
	if _, err := TaskLogDelByTaskId(task.Id); err != nil {
		t.Fatal(err)
	}
	summary, err = CurrentFailedTasks()
	if err != nil || summary.Count != 1 {
		t.Fatal("log cleanup discarded current failure")
	}
	record(TASK_SUCCESS, "", "")
	summary, err = CurrentFailedTasks()
	if err != nil || summary.Count != 0 {
		t.Fatal("recovery after cleanup")
	}

	// Test upgrading an existing schema and backfilling highest retained log ID.
	execSQL("CREATE TABLE `" + prefix + "legacy_task` LIKE `" + prefix + "task`")
	execSQL("CREATE TABLE `" + prefix + "legacy_task_log` LIKE `" + prefix + "task_log`")
	execSQL("ALTER TABLE `" + prefix + "legacy_task` DROP INDEX idx_last_status, DROP COLUMN last_log_id, DROP COLUMN last_status, DROP COLUMN last_run_time, DROP COLUMN last_error")
	execSQL("ALTER TABLE `" + prefix + "legacy_task_log` DROP INDEX idx_status_id")
	execSQL("INSERT INTO `" + prefix + "legacy_task` (id,task_name,command,notify_email) VALUES (1,'legacy','echo','')")
	execSQL("INSERT INTO `" + prefix + "legacy_task_log` (task_id,output,error,status,create_time) VALUES (1,'','old failure',-1,100),(1,'','',0,200)")
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "2026-10-21-dashboard-failures.sql"))
	if err != nil {
		t.Fatal(err)
	}
	runSQL(strings.ReplaceAll(string(migration), "`t_", "`"+prefix+"legacy_"))
	var lastStatus, lastID, lastTime int
	err = db.QueryRow("SELECT last_status,last_log_id,last_run_time FROM `"+prefix+"legacy_task` WHERE id=1").Scan(&lastStatus, &lastID, &lastTime)
	if err != nil || lastStatus != 0 || lastID != 2 || lastTime != 200 {
		t.Fatalf("migration: %d %d %d %v", lastStatus, lastID, lastTime, err)
	}
}
