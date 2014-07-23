CREATE TABLE `doctor_treatment_message` (
  `treatment_plan_id` int unsigned NOT NULL,
  `doctor_id` int unsigned NOT NULL,
  `message` text NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`treatment_plan_id`, `doctor_id`),
  FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE
) character set utf8;