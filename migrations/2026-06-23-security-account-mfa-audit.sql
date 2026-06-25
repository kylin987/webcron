ALTER TABLE `t_user`
  ADD COLUMN `mfa_secret` varchar(64) NOT NULL DEFAULT '' COMMENT 'TOTP secret' AFTER `salt`,
  ADD COLUMN `mfa_enabled` tinyint(4) NOT NULL DEFAULT '0' COMMENT '1 enabled, 0 not enabled' AFTER `mfa_secret`,
  ADD COLUMN `mfa_confirmed_at` int(11) unsigned NOT NULL DEFAULT '0' COMMENT 'MFA confirmation time' AFTER `mfa_enabled`,
  ADD COLUMN `create_time` int(11) unsigned NOT NULL DEFAULT '0' COMMENT '创建时间' AFTER `status`;

CREATE TABLE `t_operation_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL DEFAULT '0' COMMENT '操作者ID',
  `user_name` varchar(50) NOT NULL DEFAULT '' COMMENT '操作者用户名',
  `action` varchar(50) NOT NULL DEFAULT '' COMMENT '操作类型',
  `target_type` varchar(50) NOT NULL DEFAULT '' COMMENT '对象类型',
  `target_id` int(11) NOT NULL DEFAULT '0' COMMENT '对象ID',
  `result` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0成功 -1失败',
  `ip` varchar(45) NOT NULL DEFAULT '' COMMENT 'IP',
  `user_agent` varchar(255) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `request_method` varchar(10) NOT NULL DEFAULT '' COMMENT '请求方法',
  `request_path` varchar(255) NOT NULL DEFAULT '' COMMENT '请求路径',
  `detail` mediumtext NOT NULL COMMENT 'JSON详情',
  `create_time` int(11) unsigned NOT NULL DEFAULT '0' COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_time` (`user_id`, `create_time`),
  KEY `idx_action_time` (`action`, `create_time`),
  KEY `idx_target` (`target_type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
