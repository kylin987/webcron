-- 已有数据库升级时执行一次；自定义 db.prefix 时替换 t_。
-- 大日志表创建索引与回填可能耗时，建议在更新程序前的维护窗口执行。
ALTER TABLE `t_task`
 ADD COLUMN `last_log_id` bigint(20) unsigned NOT NULL DEFAULT '0',
 ADD COLUMN `last_status` tinyint(4) NOT NULL DEFAULT '0',
 ADD COLUMN `last_run_time` int(11) unsigned NOT NULL DEFAULT '0',
 ADD COLUMN `last_error` varchar(300) NOT NULL DEFAULT '',
 ADD KEY `idx_last_status` (`last_status`, `id`);
ALTER TABLE `t_task_log` ADD KEY `idx_status_id` (`status`, `id`);

-- 回填每个任务最近完成的日志；已有更新结果时不覆盖。
UPDATE `t_task` t
JOIN (SELECT task_id, MAX(id) AS log_id FROM `t_task_log` GROUP BY task_id) latest ON latest.task_id=t.id
JOIN `t_task_log` l ON l.id=latest.log_id
SET t.last_log_id=l.id, t.last_status=l.status, t.last_run_time=l.create_time, t.last_error=LEFT(l.error,300)
WHERE t.last_log_id<l.id;
