declare var Spruce: {
	BaseStaticURL: string;
	Environment: string;
	AccountPermissions: { [key: string]: bool };
	Account: {
		email: string;
	};
};

type error = string;
