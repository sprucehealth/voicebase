namespace go carefront.thriftapi

exception InternalServerError {
	1: string message
}

exception AccessDenied {
	1: string message
}

exception OverCapacity {
	1: string message
}
