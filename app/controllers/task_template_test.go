package controllers

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lisijie/webcron/app/models"
)

func TestTaskTemplatesEscapeTagsAndScript(t *testing.T) {
	payload := `"><script>alert("x")</script>&`
	funcs := template.FuncMap{
		"urlfor":   func(s string, args ...interface{}) string { return "/task/list" },
		"str2html": func(s string) template.HTML { return template.HTML(s) },
	}
	data := map[string]interface{}{
		"groups": []*models.TaskGroup{}, "groupid": 0, "tagNames": []string{payload},
		"result": "", "tag": payload, "tagsText": payload, "pageBar": "", "task": &models.Task{TaskName: payload, Command: payload},
		"list": []map[string]interface{}{{"last_status": -1, "last_log_id": int64(1), "last_error": payload, "id": 1, "name": payload, "command": payload, "description": payload, "running": 0, "status": 0, "tags": []string{payload}}},
	}
	for _, name := range []string{"list", "add", "edit"} {
		t.Run(name, func(t *testing.T) {
			tpl, err := template.New(name + ".html").Funcs(funcs).ParseFiles(filepath.Join("..", "..", "views", "task", name+".html"))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := tpl.Execute(&out, data); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "<script>alert") {
				t.Fatal("unsafe label or script HTML")
			}
			if !strings.Contains(out.String(), "&lt;script&gt;") {
				t.Fatal("escaped content missing")
			}
		})
	}
}
