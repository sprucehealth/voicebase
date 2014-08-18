CREATE TABLE `media` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `uploaded` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `uploader_id` bigint unsigned NOT NULL,
  `mimetype` varchar(128) NOT NULL,
  `url` varchar(255) NOT NULL,
  `claimer_type` varchar(64),
  `claimer_id` bigint unsigned,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`uploader_id`) REFERENCES `person` (`id`)
