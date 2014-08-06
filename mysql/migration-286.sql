CREATE TABLE `care_provider_profile` (
    `account_id` int unsigned NOT NULL,
    `full_name` varchar(250) not null,
    `why_spruce` text not null,
    `qualifications` text not null,
    `medical_school` text not null,
    `residency` text not null,
    `fellowship` text not null,
    `experience` text not null,
    `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `modified_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`account_id`),
    FOREIGN KEY (`account_id`) REFERENCES `account` (`id`) ON DELETE CASCADE
) character set utf8;
