-- 升级现有数据库时先执行；如 db.prefix 不是 t_，请替换表名前缀。
CREATE TABLE IF NOT EXISTS `t_task_tag` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `task_id` int(11) unsigned NOT NULL COMMENT '任务ID',
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '标签名称',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_task_name` (`task_id`, `name`),
  KEY `idx_name_task` (`name`, `task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
