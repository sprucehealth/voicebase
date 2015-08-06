/* @flow */

var React = require("react/addons");
var Reflux = require('reflux');
var Utils = require("../../libs/utils.js");
var Formatter = require('../../libs/formatter.js');

var Analytics = require("../../libs/analytics.js");
var AnalyticsScreenName = "demographics"
var Constants = require("./Constants.js");

var SubmitButtonView = require("./SubmitButtonView.js");
var ParentalConsentActions = require('./ParentalConsentActions.js')
var ParentalConsentStore = require('./ParentalConsentStore.js');

var IsAndroid = navigator.userAgent.indexOf('Android') >= 0;

// Based on: https://developer.mozilla.org/en-US/docs/Browser_detection_using_the_user_agent
var IsMobileSafari = navigator.userAgent.indexOf('Chrome') == 0 && navigator.userAgent.indexOf('Chromium') == 0 && navigator.userAgent.indexOf('Safari') >= 0 && navigator.userAgent.indexOf('iPhone') >= 0;

var DemographicsView = React.createClass({displayName: "DemographicsView",

	//
	// React
	//
	mixins: [
		React.addons.LinkedStateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
		Reflux.listenTo(ParentalConsentActions.saveDemographics.completed, 'saveDemographicsCompleted'),
		Reflux.listenTo(ParentalConsentActions.saveDemographics.failed, 'saveDemographicsFailed'),
	],
	propTypes: {
		onFormSubmit: React.PropTypes.func.isRequired,
	},
	getInitialState: function() {
		return {
			submitButtonPressedOnce: false,
		}
	},
	componentDidMount: function() {
		Analytics.record(AnalyticsScreenName + "_viewed", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName}, true)

		var store: ParentalConsentStoreType = this.state.store
		var userInputDemographics: ParentalConsentDemographics = store.userInput.demographics
		if (userInputDemographics) {
			this.setState({
				firstName: userInputDemographics.first_name,
				lastName: userInputDemographics.last_name,
				dob: userInputDemographics.dob,
				gender: userInputDemographics.gender,
				stateOfResidence: userInputDemographics.state,
				phone: userInputDemographics.mobile_phone,
			});
		}
		if (store.userInput.relationship) {
			this.setState({
				relationship: store.userInput.relationship,
			});
		}

		var patterns = [
			{"*": "{{99}}-{{99}}-{{9999}}"},
			{".{8}": "{{99}}-{{99}}-{{99}}"}
		]
		if (!IsAndroid && !IsMobileSafari) {
			// Note: there is likely something fundamentally wrong with this, because
			// React does not guarantee that a node we fetch here will always exist in the dom as the DOB and Phone Inputs
			// This may be why it doesn't work on Android Browser
			// We could try setting this up in componentDidUpdate, but we'll need to tear down the old instance first
			
			var dobInputFormatter = new Formatter.Formatter(React.findDOMNode(this.refs.dobInput), {
				'patterns': patterns,
				'changeCallback': this.onDateChange
			});
		}

		if (!IsAndroid) {
			var phoneInputFormatter = new Formatter.Formatter(React.findDOMNode(this.refs.phoneInput), {
				'pattern': '{{999}}-{{999}}-{{9999}}',
				'changeCallback': this.onPhoneChange
			});
		}
	},

	//
	// User interaction callbacks
	//
	handleSubmit: function(e: any) {
		e.preventDefault();
		this.setState({submitButtonPressedOnce: true})
		if (this.state.store.parentAccount.isSignedIn) {
			this.props.onFormSubmit({})
			return
		} else if (this.shouldAllowSubmit()) {
			var demographics: ParentalConsentDemographics = {
				first_name: this.state.firstName,
				last_name: this.state.lastName,
				dob: this.state.dob,
				gender: this.state.gender,
				state: this.state.stateOfResidence,
				mobile_phone: this.state.phone,
			}
			ParentalConsentActions.saveRelationship(this.state.relationship)
			ParentalConsentActions.saveDemographics(demographics)
			Analytics.record(AnalyticsScreenName + "_submission_succeeded", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName})
		} else {
			Analytics.record(AnalyticsScreenName + "_submission_failed", {
				"app_type": Constants.AnalyticsAppType,
				"screen_id": AnalyticsScreenName, 
				"error": "didn't pass client-side validation",
				"isFirstNameFieldValid": this.isFirstNameFieldValid(),
				"isLastNameFieldValid": this.isLastNameFieldValid(),
				"isDOBFieldValid": this.isDOBFieldValid(),
				"isGenderFieldValid": this.isGenderFieldValid(),
				"isRelationshipFieldValid": this.isRelationshipFieldValid(),
				"isStateOfResidenceFieldValid": this.isStateOfResidenceFieldValid(),
				"isPhoneFieldValid": this.isPhoneFieldValid(),
			})
		}
	},
	onDateChange: function(newValue: string) {
		this.setState({dob: newValue})
	},
	onPhoneChange: function(newValue: string) {
		this.setState({phone: newValue})
	},

	//
	// Action callbacks
	//
	saveDemographicsCompleted: function() {
		this.props.onFormSubmit({})
	},
	saveDemographicsFailed: function(err: ajaxError) {
		alert(err.message)
	},

	//
	// Internal
	//
	shouldAllowSubmit: function(): bool {
		if (this.state.store.parentAccount.isSignedIn) {
			return true
		}

		return this.isFirstNameFieldValid()
			&& this.isLastNameFieldValid()
			&& this.isDOBFieldValid()
			&& this.isGenderFieldValid()
			&& this.isRelationshipFieldValid()
			&& this.isStateOfResidenceFieldValid()
			&& this.isPhoneFieldValid()
	},
	isFirstNameFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.firstName)
	},
	isLastNameFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.lastName)
	},
	isDOBFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.dob)
	},
	isGenderFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.gender)
	},
	isRelationshipFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.relationship)
	},
	isStateOfResidenceFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.stateOfResidence)
	},
	isPhoneFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.phone)
	},

	render: function(): any {

		var isSignedIn = this.state.store.parentAccount.isSignedIn

		var selectContainerStyle = {
			backgroundImage: "url(" + Utils.staticURL("/img/pc/select_arrow@2x.png") + ")",
			backgroundRepeat: "no-repeat",
			backgroundSize: "12px 7px",
			backgroundPosition: "right",
		};

		var dateInput = null
		var phoneInput = null
		if (IsAndroid || IsMobileSafari) {
			dateInput = (
				<div style={{position: "relative"}}>
					<div style={{
						position: "absolute",
						lineHeight: "56px",
						fontSize: "16px",
						color: "RGBA(30, 51, 58, 0.4)",
						marginLeft: (IsMobileSafari ? "7px" : null),
					}}>
						{(Utils.isEmpty(this.state.dob) ? "Date of Birth" : null)}
					</div>
					<input 
						type="date" 
						disabled={isSignedIn}
						valueLink={this.linkState('dob')} 
						style={Utils.mergeProperties({
							height: "56px",
							width: "100%",
							border: "none",
						}, selectContainerStyle)}/>
				</div>);
			phoneInput = (<input 
				type="tel"
				disabled={isSignedIn}
				mozactionhint="done"
				inputmode="tel"
				placeholder="Mobile Phone #"
				valueLink={this.linkState('phone')}
				ref="phoneInput" />)
		} else {
			dateInput = (<input
				disabled={isSignedIn}
				type="text"
				placeholder="Date of Birth (MM-DD-YY)"
				className={this.isDOBFieldValid() ? null : "emptyState"}
				valueLink={this.linkState('dob')}
				ref="dobInput" />)
			phoneInput = (<input 
				disabled={isSignedIn}
				type="tel"
				mozactionhint="done"
				inputmode="tel"
				placeholder="Mobile Phone #"
				valueLink={this.linkState('phone')}
				ref="phoneInput" />)
		}

		var orangeBottomDividerStyle = {
			borderBottomColor: "#F5A623",
			borderBottomWidth: "2px",
		}

		var firstNameHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isFirstNameFieldValid() : false)
		var lastNameHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isLastNameFieldValid() : false)
		var dobHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isDOBFieldValid() : false)
		var genderHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isGenderFieldValid() : false)
		var relationshipHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isRelationshipFieldValid() : false)
		var stateOfResidenceHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isStateOfResidenceFieldValid() : false)
		var phoneHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isPhoneFieldValid() : false)

		var statesOfResidence = Utils.states
		statesOfResidence[0] = {name: "State of Residence", value: ""}
		var stateOfResidenceOptions: Array<any> = statesOfResidence.map(function(state: any): any {
			return (<option value={state.value} key={state.value}>{state.name}</option>)
		});

		return (
			<form
				onSubmit={this.handleSubmit}
				autoComplete="off"
				style={{
					marginTop: "8",
					fontSize: "14px",
					lineHeight: "17px",
				}}>
				<div style={{
					textAlign: "center",
					marginBottom: "22px",
				}}>
					<a href={"/login?next=%2Fpc%2F" + ParentalConsentHydration.ChildDetails.patientID + "%2Fconsent"} onClick={function (e: any) {
						// Warning: this is a synchronous request
						Analytics.record(AnalyticsScreenName + "_sign_in_link_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName})
					}}>Sign in to an existing Spruce account.</a>
				</div>
				<div className="formFieldRow hasBottomDivider hasTopDivider" style={firstNameHighlighted ? orangeBottomDividerStyle : null}>
					<input type="text"
						disabled={isSignedIn}
						autoCapitalize={IsMobileSafari ? null : "words"} // for some reason autoCapitalized was causing the first _two_ letters to be capitalized on Mobile Safari iOS 8.4
						mozactionhint="next"
						autoCorrect="on"
						inputmode="latin-name"
						placeholder="Your First Name"
						valueLink={this.linkState('firstName')} />
				</div>
				<div className="formFieldRow hasBottomDivider" style={lastNameHighlighted ? orangeBottomDividerStyle : null}>
					<input type="text"
						disabled={isSignedIn}
						autoCapitalize={IsMobileSafari ? null : "words"} // for some reason autoCapitalized was causing the first _two_ letters to be capitalized on Mobile Safari iOS 8.4
						mozactionhint="next"
						autoCorrect="on"
						inputmode="latin-name"
						placeholder="Your Last Name"
						valueLink={this.linkState('lastName')} />
					</div>
				<div className="formFieldRow hasBottomDivider" style={dobHighlighted ? orangeBottomDividerStyle : null}>
					{dateInput}
				</div>
				<div className="formFieldRow hasBottomDivider" style={Utils.mergeProperties(selectContainerStyle, genderHighlighted ? orangeBottomDividerStyle : null)}>
					<select
						disabled={isSignedIn}
						className={this.isGenderFieldValid() ? null : "emptyState"}
						name="gender"
						defaultValue=""
						valueLink={this.linkState('gender')}>
						<option value="">Gender</option>
						<option value="male">Male</option>
						<option value="female">Female</option>
					</select>
				</div>
				<div className="formFieldRow hasBottomDivider" style={Utils.mergeProperties(selectContainerStyle, relationshipHighlighted ? orangeBottomDividerStyle : null)}>
					<select
						disabled={isSignedIn}
						className={this.isRelationshipFieldValid() ? null : "emptyState"}
						defaultValue=""
						valueLink={this.linkState('relationship')}>
						<option value="">Relationship to Child</option>
						<option value="mother">Mother</option>
						<option value="father">Father</option>
						<option value="other">Other Legal Guardian</option>
					</select>
				</div>
				<div className="formFieldRow hasBottomDivider" style={Utils.mergeProperties(selectContainerStyle, stateOfResidenceHighlighted ? orangeBottomDividerStyle : null)}>
					<select
						disabled={isSignedIn}
						className={this.isStateOfResidenceFieldValid() ? null : "emptyState"}
						defaultValue=""
						valueLink={this.linkState('stateOfResidence')}>
						{stateOfResidenceOptions}
					</select>
				</div>
				<div className="formFieldRow hasBottomDivider" style={Utils.mergeProperties(phoneHighlighted ? orangeBottomDividerStyle : null, {height: "56px"})}>
					{phoneInput}
				</div>
				<div>
					<SubmitButtonView
						title="NEXT"
						appearsDisabled={!this.shouldAllowSubmit()}/>
				</div>
			</form>
		);
	}
});

module.exports = DemographicsView;