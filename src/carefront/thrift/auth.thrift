namespace go carefront.thrift.api

include "common.thrift"

struct AuthResponse {
	1: required string token
	2: required i64 account_id
}

struct TokenValidationResponse {
	1: required bool is_valid
	2: optional i64 account_id
	3: optional string reason
}

exception NoSuchLogin {
}

exception NoSuchAccount {
}

exception InvalidPassword {
	1: required i64 account_id
}

exception LoginAlreadyExists {
	1: required i64 account_id
}

service Auth {
	AuthResponse sign_up(
		1: required string login,
		2: required string password
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity,
		4: LoginAlreadyExists already_exists,
		5: InvalidPassword invalid_password)

	AuthResponse log_in(
		1: required string login,
		2: required string password
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity,
		4: NoSuchLogin no_such_login,
		5: InvalidPassword invalid_password)

	void log_out(
		1: required string token,
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity)

	TokenValidationResponse validate_token(
		1: required string token,
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity)

	void set_password(
		1: required i64 account_id,
		2: required string password
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity,
		4: NoSuchAccount no_such_account,
		5: InvalidPassword invalid_password)
}
