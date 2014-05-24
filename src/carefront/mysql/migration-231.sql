create table communication_preference (
    id int unsigned not null auto_increment,
    account_id int unsigned not null,
    communication_type varchar(50) not null, 
    creation_date timestamp(6) not null default current_timestamp(6),
    status varchar(100) not null, 
    primary key (id),
    foreign key (account_id) references account(id),
    unique key (account_id, communication_type)
) character set utf8;

create table push_config (
    id int unsigned not null auto_increment,
    account_id int unsigned not null,
    device_token varbinary(500) not null,
    push_endpoint varchar(300) not null,
    platform varchar(100) not null,
    platform_version varchar(100) not null,
    app_type varchar(100) not null,
    app_env varchar(100) not null,
    app_version varchar(100) not null,
    device varchar(100) not null,
    device_model varchar(100) not null,
    device_id varchar(100) not null,
    creation_date timestamp(6) not null default current_timestamp(6),
    primary key (id),
    foreign key (account_id) references account(id),
    unique key (device_token)
) character set utf8;

create table patient_prompt_status (
  id int unsigned not null auto_increment,
  prompt_status varchar(100) not null,
  patient_id int unsigned not null,
  creation_date timestamp(6) not null default current_timestamp(6),
  primary key (id),
  foreign key (patient_id) references patient(id)
) character set utf8;