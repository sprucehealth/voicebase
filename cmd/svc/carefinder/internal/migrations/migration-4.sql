-- add the following two columns to store the referral code and the link.
-- the direct link is expected to contain the code. 
-- the code is used to generate a valid link to send to the user.
ALTER TABLE carefinder_doctor_info ADD COLUMN referral_code TEXT;
ALTER TABLE carefinder_doctor_info ADD COLUMN referral_link TEXT;