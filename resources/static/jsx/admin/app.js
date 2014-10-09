/** @jsx React.DOM */

var Perms = require("./permissions.js");
var Nav = require("../nav.js");

var Accounts = require("./accounts.js");
var Analytics = require("./analytics.js");
var Dashboard = require("./dashboard.js");
var Doctors = require("./doctors.js");
var Drugs = require("./drugs.js");
var Email = require("./email.js");
var Guides = require("./guides.js");

window.AdminRouter = Backbone.Router.extend({
	routes : {
		"": function() {
			this.current = "dashboard";
			this.params = {};
		},
		"doctors": function() {
			this.current = "doctorSearch";
			this.params = {};
		},
		"doctors/:doctorID/:page": function(doctorID, page) {
			this.current = "doctor";
			this.params = {doctorID: doctorID, page: page};
		},
		"guides/:page": function(page) {
			this.current = "guides";
			this.params = {page: page};
		},
		"guides/:page/:id": function(page, guideID) {
			this.current = "guides";
			this.params = {page: page, guideID: guideID};
		},
		"analytics/:page": function(page) {
			this.current = "analytics";
			this.params = {page: page};
		},
		"analytics/:page/:id": function(page, reportID) {
			this.current = "analytics";
			this.params = {page: page, reportID: reportID};
		},
		"accounts": function() {
			this.current = "accountsList";
			this.params = {};
		},
		"accounts/:accountID/:page": function(accountID, page) {
			this.current = "account";
			this.params = {accountID: accountID, page: page};
		},
		"email": function() {
			this.current = "email";
			this.params = {};
		},
		"email/:typeKey": function(typeKey) {
			this.current = "email";
			this.params = {typeKey: typeKey};
		},
		"email/:typeKey/:templateID": function(typeKey, templateID) {
			this.current = "email";
			this.params = {typeKey: typeKey, templateID: templateID};
		},
		"email/:typeKey/:templateID/edit": function(typeKey, templateID) {
			this.current = "email";
			this.params = {typeKey: typeKey, templateID: templateID, edit: true};
		},
		"drugs": function() {
			this.current = "drugs";
			this.params = {};
		}
	}
});

window.Admin = React.createClass({displayName: "Admin",
	getDefaultProps: function() {
		var leftMenuItems = [
			{
				id: "dashboard",
				url: "",
				name: "Dashboard"
			}
		];

		if (Perms.has(Perms.DoctorsView)) {
			leftMenuItems.push({
				id: "doctorSearch",
				url: "doctors",
				name: "Doctors"
			});
		};

		leftMenuItems.push({
				id: "guides",
				url: "guides/resources",
				name: "Guides"
		});

		if (Perms.has(Perms.AnalyticsReportsView)) {
			leftMenuItems.push({
				id: "analytics",
				url: "analytics/query",
				name: "Analytics"
			});
		}

		if (Perms.has(Perms.EmailView)) {
			leftMenuItems.push({
				id: "email",
				url: "email",
				name: "Email"
			});
		}

		if (Perms.has(Perms.AdminAccountsView)) {
			leftMenuItems.push({
				id: "accounts",
				url: "accounts",
				name: "Accounts"
			});
		}

		leftMenuItems.push({
			id: "drugs",
			url: "drugs",
			name: "Drugs"
		});

		var rightMenuItems = [];

		return {
			leftMenuItems: leftMenuItems,
			rightMenuItems: rightMenuItems
		}
	},
	dashboard: function() {
		return <div className="container-fluid main"><Dashboard.Dashboard router={this.props.router} /></div>;
	},
	doctorSearch: function() {
		return <div className="container-fluid main"><Doctors.DoctorSearch router={this.props.router} /></div>;
	},
	doctor: function() {
		return <Doctors.Doctor router={this.props.router} doctorID={this.props.router.params.doctorID} page={this.props.router.params.page} />;
	},
	guides: function() {
		return <Guides.Guides router={this.props.router} page={this.props.router.params.page} guideID={this.props.router.params.guideID} />;
	},
	analytics: function() {
		return <Analytics.Analytics router={this.props.router} page={this.props.router.params.page} reportID={this.props.router.params.reportID} />;
	},
	email: function() {
		return <Email.EmailAdmin router={this.props.router} typeKey={this.props.router.params.typeKey} templateID={this.props.router.params.templateID} edit={this.props.router.params.edit} />;
	},
	accountsList: function() {
		return <Accounts.AccountList router={this.props.router} accountID={this.props.router.params.accountID} />;
	},
	account: function() {
		return <Accounts.Account router={this.props.router} accountID={this.props.router.params.accountID} page={this.props.router.params.page} />;
	},
	drugs: function() {
		return <Drugs.DrugSearch router={this.props.router} accountID={this.props.router.params.accountID} />;
	},
	componentWillMount : function() {
		this.callback = (function() {
			this.forceUpdate();
		}).bind(this);
		this.props.router.on("route", this.callback);
	},
	componentWillUnmount : function() {
		this.props.router.off("route", this.callback);
	},
	render: function() {
		var page = this[this.props.router.current];
		return (
			<div>
				<Nav.TopNav router={this.props.router} leftItems={this.props.leftMenuItems} rightItems={this.props.rightMenuItems} activeItem={this.props.router.current} name="Admin" />
				{page ? page() : "Page Not Found"}
			</div>
		);
	}
});
