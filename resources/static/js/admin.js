var AdminAPI = {
	// cb is function(success: bool, data: object, jqXHR: jqXHR)
	ajax: function(params, cb) {
		params.success = function(data) {
			cb(true, data, null);
		}
		params.error = function(jqXHR) {
			cb(false, null, jqXHR);
		}
		params.url = "/admin/api" + params.url;
		jQuery.ajax(params);
	},

	medicalLicenses: function(doctorID, cb) {
		this.ajax({
			type: "GET",
			url: "/doctor/" + doctorID + "/licenses",
			dataType: "json"
		}, cb);
	},

	careProviderProfile: function(doctorID, cb) {
		this.ajax({
			type: "GET",
			url: "/doctor/" + doctorID + "/profile",
			dataType: "json"
		}, cb);
	},

	updateCareProviderProfile: function(doctorID, profile, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor/" + doctorID + "/profile",
			data: JSON.stringify(profile),
			dataType: "json"
		}, cb);
	}
};

var AdminNav = React.createClass({displayName: "AdminNav",
	componentWillMount : function() {
		this.callback = (function() {
			this.forceUpdate();
		}).bind(this);
		this.props.router.on("route", this.callback);
	},
	componentWillUnmount : function() {
		this.props.router.off("route", this.callback);
	},
	navigate: function(path) {
		this.props.router.navigate(path, {
			trigger : true
		});
		return false;
	},
	render: function() {
		var t = this;
		var pageComponent;
		function createItem(item) {
			var cls = "";
			if (item.id == t.props.router.current) {
				cls = "active"
				pageComponent = item.component;
			}
			return React.DOM.li({key: item.id, onClick: t.navigate.bind(null, item.id), className: cls},
				React.DOM.a({href: "#"}, item.name)
			);
		}
		var navItems = this.props.items.map(createItem);
		return React.DOM.div({className: "container-fluid"},
			React.DOM.div({className: "row"},
				React.DOM.div({className: "col-sm-3 col-md-2 sidebar"},
					React.DOM.ul({className: "nav nav-sidebar"},
						navItems
					)
				)
			),
			React.DOM.div({className: "col-sm-9 col-sm-offset-3 col-md-10 col-md-offset-2 main"},
				pageComponent
			)
		);
	}
});

var FormInput = React.createClass({displayName: "FormInput",
	getDefaultProps: function() {
		return {type: "text"}
	},
	render: function() {
		return React.DOM.div({className: "form-group"},
			React.DOM.label({
				className: "control-label",
				htmlFor: this.props.id
			}, this.props.label),
			React.DOM.input({
				type: this.props.type,
				className: "form-control section-name",
				name: this.props.id,
				value: this.props.value,
				onChange: this.props.onChange
			})
		);
	}
});

var TextArea = React.createClass({displayName: "TextArea",
	getDefaultProps: function() {
		return {rows: 5}
	},
	render: function() {
		return React.DOM.div({className: "form-group"},
			React.DOM.label({
				className: "control-label",
				htmlFor: this.props.id
			}, this.props.label),
			React.DOM.textarea({
				type: "text",
				className: "form-control section-name",
				name: this.props.id,
				value: this.props.value,
				rows: this.props.rows,
				onChange: this.props.onChange
			})
		);
	}
});

var Alert = React.createClass({displayName: "Alert",
	propTypes: {
		type: React.PropTypes.oneOf(['success', 'info', 'warning', 'danger'])
	},
	getDefaultProps: function() {
		return {"type": "info"};
	},
	render: function() {
		if (this.props.children.length == 0) {
			return null;
		}
		return React.DOM.div({className: "alert alert-"+this.props.type, role: "alert"}, this.props.children);
	}
});
