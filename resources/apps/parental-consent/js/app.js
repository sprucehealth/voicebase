/* @flow */

var Backbone = require("backbone");
var jQuery = require("jquery");
var React = require("react");
var Routing = require("../../libs/routing.js");
var Reflux = require('reflux');
var Utils = require("../../libs/utils.js");

var ParentalConsentStore = require('./ParentalConsentStore.js');
var TitleView = require("./TitleView.js")
var DemographicsView = require("./DemographicsView.js")
var ContentContainer = require("./ContentContainer.js")
var EmailRelationshipConsentView = require("./EmailRelationshipConsentView.js")
var PhotoIdentificationView = require("./PhotoIdentificationView.js")
var ConfirmationView = require("./ConfirmationView.js")
window.FAQ = require("./faq.js");

window.React = React; // export for http://fb.me/react-devtools
Backbone.jQuery = jQuery;

var calculateNumSectionsForStore = function(store: ParentalConsentStoreType): number {
	var numSections = 4
	if (store.parentAccount && store.parentAccount.WasSignedInAtPageLoad) {
		console.log("removing section due to WasSignedInAtPageLoad")
		numSections = numSections - 1
	}
	if (store.parentAccount && store.parentAccount.ParentalConsent && store.parentAccount.ParentalConsent.consented) {
		console.log("removing section due to PhotoIdentificationAlreadySubmittedAtPageLoad")
		numSections = numSections - 1
	}
	if (store.PhotoIdentificationAlreadySubmittedAtPageLoad) {
		console.log("removing section due to PhotoIdentificationAlreadySubmittedAtPageLoad")
		numSections = numSections - 1
	}
	return numSections
}

var sectionRoutesForStore = function(store: ParentalConsentStoreType): Array<string> {
	var sections = []
	if (!store.parentAccount || !store.parentAccount.WasSignedInAtPageLoad) {
		console.log("adding demographics due to !WasSignedInAtPageLoad")
		sections.push("demographics")
	}
	if (!store.ConsentWasAlreadySubmittedAtPageLoad) {
		sections.push("consent")
	}
	if (!store.PhotoIdentificationAlreadySubmittedAtPageLoad) {
		console.log("adding identification due to !PhotoIdentificationAlreadySubmittedAtPageLoad")
		sections.push("identification")
	}
	sections.push("confirmation")
	return sections
}

var routeForSectionIndexAndStore = function(sectionIndex: number, store: ParentalConsentStoreType): string {
	var routes: Array<string> = sectionRoutesForStore(store)
	return routes[sectionIndex]
}

var sectionIndexForRouteAndStore = function(route: string, store: ParentalConsentStoreType): number {
	var sections = sectionRoutesForStore(store)
	var index = sections.indexOf(route)
	if (index === -1) {
		console.log("index of section not found")
	}
	return index
}

var nextRouteAfterRouteForStore = function(currentRoute: string, store: ParentalConsentStoreType): string {
	var nextRoute: string = ""
	var index: number = sectionIndexForRouteAndStore(currentRoute, store)
	if (index !== -1) {
		nextRoute = routeForSectionIndexAndStore(index + 1, store)
	}
	if (Utils.isEmpty(nextRoute)) {
		console.log("something's gone wrong and we can't determine the next route")
	}
	return nextRoute
}

var ParentalConsentRouter = Backbone.Router.extend({
	routes : {
		"": function() {
			this.current = routeForSectionIndexAndStore(0, ParentalConsentStore.getCurrentState());
			this.params = {};
		},
		"demographics": function() {
			this.current = "demographics";
			this.params = {};
		},
		"consent": function() {
			this.current = "consent";
			this.params = {};
		},
		"identification": function() {
			this.current = "photoIDs";
			this.params = {};
		},
		"confirmation": function() {
			this.current = "confirmation";
			this.params = {};
		},
	}
});

var ParentalConsent = React.createClass({displayName: "ParentalConsent",
	pages: {
		demographics: function() {
			return <DemographicsPage router={this.props.router} />
		},
		consent: function() {
			return <EmailAndConsentPage router={this.props.router} />
		},
		photoIDs: function() {
			return <PhotoIdentificationPage router={this.props.router} />
		},
		confirmation: function() {
			return <ConfirmationPage router={this.props.router} />
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
	render: function(): any {
		var page = this.pages[this.props.router.current];
		return (
			<div>{page ? page.bind(this)() : "Page Not Found"}</div>
		);
	}
});



var DemographicsPage = React.createClass({displayName: "DemographicsPage",
	mixins: [
		Routing.RouterNavigateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
	],
	handleSubmit: function() {
		var currentRoute = "demographics"
		var nextRoute: string = nextRouteAfterRouteForStore(currentRoute, this.state.store)
		this.props.router.navigate(nextRoute, {trigger: true});
	},
	render: function(): any {
		return (
			<ContentContainer
				busy={false}
				showSectionedProgressBar={true}
				currentSectionIndex={0}
				numSections={calculateNumSectionsForStore(this.state.store)}
				content={(
					<div>
						<TitleView
							title={"Authorization for " + this.state.store.childDetails.firstName + "'s Visit"}
							subtitle="First we need to know some basic information about you."
							text="" />
						<DemographicsView
							onFormSubmit={this.handleSubmit} />
					</div>
				)} />
		);
	}
});

var EmailAndConsentPage = React.createClass({displayName: "EmailAndConsentPage",
	mixins: [
		Routing.RouterNavigateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
	],
	handleSubmit: function() {
		var currentRoute = "consent"
		var nextRoute: string = nextRouteAfterRouteForStore(currentRoute, this.state.store)
		this.props.router.navigate(nextRoute, {trigger: true});
	},
	getInitialState: function() {
		return {}
	},
	render: function(): any {
		var store: ParentalConsentStoreType = this.state.store
		var t = this
		return (
			<ContentContainer
				busy={store.numBlockingOperations > 0}
				showSectionedProgressBar={true}
				currentSectionIndex={1}
				numSections={calculateNumSectionsForStore(this.state.store)}
				content={(
					<div>
						<TitleView
							title={"Authorization for " + this.state.store.childDetails.firstName + "'s Visit"}
							subtitle="Now create your Spruce account so you can log in and view your child’s care record."
							text="" />
						<EmailRelationshipConsentView
							collectEmailAndPassword={!store.parentAccount.WasSignedInAtPageLoad}
							collectRelationship={store.parentAccount.WasSignedInAtPageLoad}
							onFormSubmit={t.handleSubmit} />
					</div>
				)} />
		);
	}
});

var PhotoIdentificationPage = React.createClass({displayName: "PhotoIdentificationPage",
	mixins: [
		Routing.RouterNavigateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
	],
	handleSubmit: function() {
		var currentRoute = "identification"
		var nextRoute: string = nextRouteAfterRouteForStore(currentRoute, this.state.store)
		this.props.router.navigate(nextRoute, {trigger: true});
	},
	render: function(): any {
		return (
			<ContentContainer
				busy={false}
				showSectionedProgressBar={true}
				currentSectionIndex={2}
				numSections={calculateNumSectionsForStore(this.state.store)}
				content={(
					<div>
						<TitleView
							title={"Authorization for " + this.state.store.childDetails.firstName + "'s Visit"}
							subtitle="Upload a photo of your government issued photo ID."
							text="To protect the safety of minors on Spruce, we need to be confident that adults responsible for them are of age to consent to treatment." />
						<PhotoIdentificationView
							onFormSubmit={this.handleSubmit} />
					</div>
				)} />
		);
	}
});

var ConfirmationPage = React.createClass({displayName: "ConfirmationPage",
	mixins: [
		Routing.RouterNavigateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
	],
	render: function(): any {
		return (
			<ContentContainer
				busy={false}
				showSectionedProgressBar={false}
				content={(
					<div>
						<ConfirmationView
							onFormSubmit={this.handleSubmit} />
					</div>
				)} />
		);
	}
});

jQuery(function() {
	var router = new ParentalConsentRouter();
	router.root = "/parental-consent/";
	React.render(React.createElement(ParentalConsent, {
		router: router
	}), document.getElementById('base-content'));
	Backbone.history.start({
		pushState: true,
		root: router.root,
	});
});
