-- Record of an RX reminder associated with a treatment
CREATE TABLE rx_reminder (
    treatment_id INT UNSIGNED NOT NULL, -- teatment still uses a normal int for key/index
    text VARCHAR(255) NOT NULL,
    reminder_interval VARCHAR(15) NOT NULL, -- Interval is a reserved word in mysql so this prefix must be maintained
    days VARCHAR(60), -- MONDAY,TUESDAY,WEDNESDAY,THURSDAY,FRIDAY,SATURDAY,SUNDAY
    times VARCHAR(8640) NOT NULL, -- Times are of the format `99:99` stored in comma seperated list and can be any possible unique time. (24*60*6)-1=8639
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (treatment_id),
    CONSTRAINT rx_reminder_treatment_id FOREIGN KEY (treatment_id) REFERENCES treatment (id));