ALTER TABLE `photo_intake_slot` DROP FOREIGN KEY `photo_intake_slot_ibfk_2`;
ALTER TABLE `photo_intake_slot` ADD FOREIGN KEY (`photo_id`) REFERENCES `media` (`id`); 