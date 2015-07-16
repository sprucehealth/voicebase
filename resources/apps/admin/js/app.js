/* @flow */

var React = require("react");
window.React = React; // export for http://fb.me/react-devtools
var Backbone = require("backbone");
Backbone.$ = window.$;

var Accounts = require("./accounts.js");
var Analytics = require("./analytics.js");
var Dashboard = require("./dashboard.js");
var Doctors = require("./doctors.js");
var FavoriteTreatmentPlan = require("./favorite_treatment_plan.js");
var Financial = require("./financial.js");
var Guides = require("./guides.js");
var Nav = require("../../libs/nav.js");
var Pathways = require("./pathways.js");
var Perms = require("./permissions.js");
var Settings = require("./settings.js");
var Visit = require("./visit.js");
var CareCoordinator = require("./care_coordinator.js");
var Marketing = require("./marketing.js");

var AdminRouter = Backbone.Router.extend({
	routes : {
		"": function() {
			this.current = "dashboard";
			this.params = {};
		},
		"careproviders": function() {
			this.current = "careProviders";
			this.params = {};
		},
		"careproviders/:page": function(page) {
			this.current = "careProviders";
			this.params = {page: page};
		},
		"careproviders/account/:doctorID/:page": function(doctorID, page) {
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
		"accounts/:accountID/:page": function(accountID, page) {
			this.current = "account";
			this.params = {accountID: accountID, page: page};
		},
		"pathways": function() {
			this.current = "pathways";
			this.params = {page: "list"};
		},
		"pathways/:page": function(page) {
			this.current = "pathways";
			this.params = {page: page};
		},
		"pathways/:page/:id": function(page, pathwayID) {
			this.current = "pathways";
			this.params = {page: page, pathwayID: pathwayID};
		},
		"financial": function() {
			this.current = "financial";
			this.params = {page: "incoming"}
		},
		"financial/:page": function(page) {
			this.current = "financial";
			this.params = {page: page};
		},
		"treatment_plan/favorite/:ftpID/:page": function(ftpID, page) {
			this.current = "favoriteTreatmentPlan";
			this.params = {page: page, ftpID: ftpID};
		},
		"case/visit": function() {
			this.current = "visit";
			this.params = {page: "overview"};
		},
		"case/:caseID/visit/:visitID": function(caseID, visitID) {
			this.current = "visit";
			this.params = {page: "details", caseID: caseID, visitID: visitID};
		},
		"settings": function() {
			this.current = "settings";
			this.params = {page: null};
		},
		"settings/:page": function(page) {
			this.current = "settings";
			this.params = {page: page};
		},
		"carecoordinator/drugs": function() {
			this.current = "carecoordinator";
			this.params = {page: "drugs"};
		},
		"carecoordinator/tags/:page": function(page) {
			this.current = "carecoordinator";
			this.params = {page: "tags_"+page};
		},
		"marketing/:page": function(page) {
			this.current = "marketing";
			this.params = {page: page};
		},
		"marketing/promotions/:page": function(page) {
			this.current = "marketing";
			this.params = {page: page};
		},
	}
});

var Admin = React.createClass({displayName: "Admin",
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
				id: "careProviders",
				url: "careproviders/search",
				name: "Care Providers"
			});
		};

		if (Perms.has(Perms.ResourceGuidesView) || Perms.has(Perms.RXGuidesView)) {
			leftMenuItems.push({
				id: "guides",
				url: "guides/resources",
				name: "Guides"
			});
		}

		if (Perms.has(Perms.AnalyticsReportsView)) {
			leftMenuItems.push({
				id: "analytics",
				url: "analytics/query",
				name: "Analytics"
			});
		}

		if (Perms.has(Perms.PathwaysView)) {
			leftMenuItems.push({
				id: "pathways",
				url: "pathways",
				name: "Pathways"
			});
		}

		if (Perms.has(Perms.FinancialView)) {
			leftMenuItems.push({
				id: "financial",
				url: "financial/incoming",
				name: "Financial"
			});
		}

		if (Perms.has(Perms.CareCoordinatorView)) {
			leftMenuItems.push({
				id: "carecoordinator",
				url: "carecoordinator/drugs",
				name: "Care Coordinator"
			});
		}

		if (Perms.has(Perms.MarketingView)) {
			leftMenuItems.push({
				id: "marketing",
				url: "marketing/promotions",
				name: "Marketing"
			});
		}

		if (Perms.has(Perms.AdminAccountsView)) {
			leftMenuItems.push({
				id: "settings",
				url: "settings/accounts",
				name: "Settings"
			});
		} else if (Perms.has(Perms.CfgView)) {
			leftMenuItems.push({
				id: "settings",
				url: "settings/cfg",
				name: "Settings"
			});
		} else if (Perms.has(Perms.AnalyticsReportsView)) {
			leftMenuItems.push({
				id: "settings",
				url: "settings/schedmsg",
				name: "Settings"
			});
		}

		var rightMenuItems = [];

		return {
			leftMenuItems: leftMenuItems,
			rightMenuItems: rightMenuItems
		}
	},
	pages: {
		dashboard: function() {
			return <div className="container-fluid main"><Dashboard.Dashboard router={this.props.router} /></div>;
		},
		careProviders: function() {
			return <Doctors.CareProvidersPage router={this.props.router} page={this.props.router.params.page} />;
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
		account: function() {
			return <Accounts.Account router={this.props.router} accountID={this.props.router.params.accountID} page={this.props.router.params.page} />;
		},
		pathways: function() {
			return <Pathways.Page router={this.props.router} page={this.props.router.params.page} pathwayID={this.props.router.params.pathwayID} />;
		},
		financial: function() {
			return <Financial.Page router={this.props.router} page={this.props.router.params.page} />;
		 },
		favoriteTreatmentPlan: function() {
			return <FavoriteTreatmentPlan.Page router={this.props.router} page={this.props.router.params.page} ftpID={this.props.router.params.ftpID} />;
		},
		visit: function() {
			return <Visit.Page router={this.props.router} page={this.props.router.params.page} caseID={this.props.router.params.caseID} visitID={this.props.router.params.visitID} />;
		},
		carecoordinator: function() {
			return <CareCoordinator.Page router={this.props.router} page={this.props.router.params.page} accountID={this.props.router.params.accountID} />;
		},
		marketing: function() {
			return <Marketing.Page router={this.props.router} page={this.props.router.params.page} />;
		},
		settings: function() {
			return <Settings.Page router={this.props.router} page={this.props.router.params.page} />;
		},
	},
	componentWillMount : function() {
		this.props.router.on("route", function() {
			this.forceUpdate();
		}.bind(this));
	},
	componentWillUnmount : function() {
		this.props.router.off("route", function() {
			this.forceUpdate();
		}.bind(this));
	},
	render: function() {
		var page = this.pages[this.props.router.current];
		return (
			<div>
				<Nav.TopNav router={this.props.router} leftItems={this.props.leftMenuItems} rightItems={this.props.rightMenuItems} activeItem={this.props.router.current} name="Admin" />
				{page ? page.bind(this)() : "Page Not Found"}
			</div>
		);
	}
});

jQuery(function() {
	var router = new AdminRouter();
	router.root = "/admin/";
	React.render(React.createElement(Admin, {
		router: router
	}), document.getElementById('base-content'));
	Backbone.history.start({
		pushState: true,
		root: router.root,
	});
});
