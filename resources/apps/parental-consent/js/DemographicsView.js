/* @flow */

var React = require("react/addons");
var Reflux = require('reflux');
var Utils = require("../../libs/utils.js");
var Formatter = require('../../libs/formatter.js');

var SubmitButtonView = require("./SubmitButtonView.js");
var ParentalConsentActions = require('./ParentalConsentActions.js')
var ParentalConsentStore = require('./ParentalConsentStore.js');

var DemographicsView = React.createClass({displayName: "DemographicsView",

	//
	// React
	//
	mixins: [
		React.addons.LinkedStateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
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

		var dobInputFormatter = new Formatter.Formatter(React.findDOMNode(this.refs.dobInput), {
			'pattern': '{{99}}-{{99}}-{{99}}',
			'changeCallback': this.onDateChange
		});

		var phoneInputFormatter = new Formatter.Formatter(React.findDOMNode(this.refs.phoneInput), {
			'pattern': '{{999}}-{{999}}-{{9999}}',
			'changeCallback': this.onPhoneChange
		});
	},

	//
	// User interaction callbacks
	//
	handleSubmit: function(e: any) {
		e.preventDefault();
		this.setState({submitButtonPressedOnce: true})
		if (this.shouldAllowSubmit()) {
			var demographics: ParentalConsentDemographics = {
				first_name: this.state.firstName,
				last_name: this.state.lastName,
				dob: this.state.dob,
				gender: this.state.gender,
				state: this.state.stateOfResidence,
				mobile_phone: this.state.phone,
			}
			ParentalConsentActions.saveRelationship(this.state.relationship)
			var t = this
			ParentalConsentActions.saveDemographics.triggerPromise(demographics).then(function(response: any) {
				t.props.onFormSubmit({})
			}).catch(function(err: ajaxError) {
				alert(err.message)
			});
		}
	},
	onDateChange: function(newValue: string) {
		this.setState({dob: newValue})
	},
	onPhoneChange: function(newValue: string) {
		this.setState({phone: newValue})
	},

	//
	// Internal
	//
	shouldAllowSubmit: function(): bool {
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
		var selectContainerStyle = {
			backgroundImage: "url(/static/img/pc/select_arrow@2x.png)",
			backgroundRepeat: "no-repeat",
			backgroundSize: "12px 7px",
			backgroundPosition: "right",
		};

		var submitButtonDisabled = !this.shouldAllowSubmit()

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
			style={{
				marginTop: "8",
				fontSize: "14px",
				lineHeight: "17px",
			}}
			autoComplete="on">
				<div style={{
					textAlign: "center",
					marginBottom: "22px",
				}}>
					<a href={"/login?next=%2Fpc%2F" + ParentalConsentHydration.ChildDetails.patientID + "%2Fconsent"}>Sign in to an existing Spruce account.</a>
				</div>
				<div className="formFieldRow hasBottomDivider hasTopDivider" style={firstNameHighlighted ? orangeBottomDividerStyle : null}>
					<input type="text"
						// autoCapitalize="words" // for some reason autoCapitalized was causing the first _two_ letters to be capitalized on Mobile Safari iOS 8.4
						mozactionhint="next"
						autoComplete="given-name"
						autoCorrect="on"
						inputmode="latin-name"
						placeholder="Your First Name"
						valueLink={this.linkState('firstName')} />
				</div>
				<div className="formFieldRow hasBottomDivider" style={lastNameHighlighted ? orangeBottomDividerStyle : null}>
					<input type="text"
						// autoCapitalize="words" // for some reason autoCapitalized was causing the first _two_ letters to be capitalized on Mobile Safari iOS 8.4
						mozactionhint="next"
						autoComplete="family-name"
						autoCorrect="on"
						inputmode="latin-name"
						placeholder="Your Last Name"
						valueLink={this.linkState('lastName')} />
					</div>
				<div className="formFieldRow hasBottomDivider" style={dobHighlighted ? orangeBottomDividerStyle : null}>
					<input
						type="text"
						placeholder="Date of Birth (MM-DD-YY)"
						className={this.isDOBFieldValid() ? null : "emptyState"}
						autoComplete="bday"
						valueLink={this.linkState('dob')}
						ref="dobInput" />
				</div>
				<div className="formFieldRow hasBottomDivider" style={Utils.mergeProperties(selectContainerStyle, genderHighlighted ? orangeBottomDividerStyle : null)}>
					<select
						className={this.isGenderFieldValid() ? null : "emptyState"}
						name="gender"
						defaultValue=""
						autoComplete="sex"
						valueLink={this.linkState('gender')}>
						<option value="">Gender</option>
						<option value="male">Male</option>
						<option value="female">Female</option>
					</select>
				</div>
				<div className="formFieldRow hasBottomDivider" style={Utils.mergeProperties(selectContainerStyle, relationshipHighlighted ? orangeBottomDividerStyle : null)}>
					<select
						className={this.isRelationshipFieldValid() ? null : "emptyState"}
						defaultValue=""
						valueLink={this.linkState('relationship')}>
						<option value="">Relationship to Child</option>
						<option value="mother">Mother</option>
						<option value="father">Father</option>
						<option value="other">Other Legal Guardian</option>
					</select>
				</div>
				<div className="formFieldRow hasBottomDivider" autoComplete="state" style={Utils.mergeProperties(selectContainerStyle, stateOfResidenceHighlighted ? orangeBottomDividerStyle : null)}>
					<select
						className={this.isStateOfResidenceFieldValid() ? null : "emptyState"}
						defaultValue=""
						valueLink={this.linkState('stateOfResidence')}>
						{stateOfResidenceOptions}
					</select>
				</div>
				<div className="formFieldRow hasBottomDivider" style={phoneHighlighted ? orangeBottomDividerStyle : null}>
					<input type="tel"
						mozactionhint="done"
						autoComplete="tel"
						inputmode="tel"
						placeholder="Mobile Phone #"
						valueLink={this.linkState('phone')}
						ref="phoneInput" />
				</div>
				<div>
					<SubmitButtonView
						title="NEXT"
						appearsDisabled={submitButtonDisabled}/>
				</div>
			</form>
		);
	}
});

module.exports = DemographicsView;