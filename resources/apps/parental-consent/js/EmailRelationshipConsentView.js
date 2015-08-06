/* @flow */

var React = require("react");
var Reflux = require('reflux')
var Utils = require("../../libs/utils.js");

var Analytics = require("../../libs/analytics.js");
var AnalyticsScreenName = "consent"

var Constants = require("./Constants.js");
var SubmitButtonView = require("./SubmitButtonView.js");
var ParentalConsentActions = require('./ParentalConsentActions.js')
var ParentalConsentStore = require('./ParentalConsentStore.js');

var IsIOS = navigator.userAgent.indexOf('iPhone') >= 0;

var EmailRelationshipConsentView = React.createClass({displayName: "EmailRelationshipConsentView",

	//
	// React
	//
	mixins: [
		React.addons.LinkedStateMixin,
		Reflux.connect(ParentalConsentStore, 'store'),
		Reflux.listenTo(ParentalConsentActions.submitEmailRelationshipConsent.completed, 'submitEmailRelationshipConsentCompleted'),
		Reflux.listenTo(ParentalConsentActions.submitEmailRelationshipConsent.failed, 'submitEmailRelationshipConsentFailed'),
	],
	propTypes: {
		collectEmailAndPassword: React.PropTypes.bool.isRequired,
		collectRelationship: React.PropTypes.bool.isRequired,
		onFormSubmit: React.PropTypes.func.isRequired,
	},
	getInitialState: function() {
		return {
			submitButtonPressedOnce: false,
			consentedToTermsAndPrivacy: false,
			consentedToUseOfTelehealth: false,
		}
	},
	componentDidMount: function() {
		Analytics.record(AnalyticsScreenName + "_viewed", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName})
		
		var store: ParentalConsentStoreType = this.state.store
		if (store.userInput && store.userInput.emailPassword) {
			this.setState({
				email: store.userInput.emailPassword.email,
				password: store.userInput.emailPassword.password,
			});
		}
		if (store.userInput && store.userInput.consents) {
			this.setState({
				consentedToTermsAndPrivacy: store.userInput.consents.consentedToTermsAndPrivacy,
				consentedToUseOfTelehealth: store.userInput.consents.consentedToConsentToUseOfTelehealth,
			});
		}
		if (store.userInput && store.userInput.relationship) {
			this.setState({
				relationship: store.userInput.relationship,
			});
		}
	},
	componentWillUnmount: function() {
		this.saveDataToStore()
	},

	//
	// Action callbacks
	//
	submitEmailRelationshipConsentCompleted: function() {
		Analytics.record(AnalyticsScreenName + "_submission_succeeded", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName})
		this.props.onFormSubmit({})
	},
	submitEmailRelationshipConsentFailed: function(error: ajaxError) {
		Analytics.record(AnalyticsScreenName + "_submission_failed", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName, "error": error.message})
		alert(error.message)
		if (error.type === "account_exists") {
			this.setState({showSignInCTA: true})
		}
	},

	//
	// User interaction callbacks
	//
	handleSubmit: function(e: any) {
		e.preventDefault();
		var t = this
		this.setState({submitButtonPressedOnce: true})

		var hasConsented = this.state.store.userInput.consents.consentedToTermsAndPrivacy && this.state.store.userInput.consents.consentedToConsentToUseOfTelehealth
		var justRegisteredOrJustConsented = (this.state.store.parentAccount.isSignedIn && !this.state.store.parentAccount.WasSignedInAtPageLoad) || hasConsented
		if (justRegisteredOrJustConsented) {
			this.props.onFormSubmit({})
		} else {
			if (this.shouldAllowSubmit()) {
				this.saveDataToStore()
				ParentalConsentActions.submitEmailRelationshipConsent()
			} else {
				var error: ajaxError = {type: "client_validation", message: "didn't pass client-side validation"}
				Analytics.record(AnalyticsScreenName + "_submission_failed", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName, "error": error.message})
			}
		}		
	},
	saveDataToStore: function() {
		if (this.props.collectEmailAndPassword) {
			var emailPassword: ParentalConsentEmailPassword = {
				email: this.state.email,
				password: this.state.password,
			}
			ParentalConsentActions.saveEmailAndPassword(emailPassword)
		}

		if (this.props.collectRelationship) {
			ParentalConsentActions.saveRelationship(this.state.relationship)
		}
	},


	//
	// Internal
	//
	shouldAllowSubmit: function(): bool {
		var hasConsented = this.state.store.consentedToTermsAndPrivacy && this.state.store.consentedToConsentToUseOfTelehealth
		var justRegisteredOrJustConsented = (this.state.store.parentAccount.isSignedIn && !this.state.store.parentAccount.WasSignedInAtPageLoad) || hasConsented
		if (justRegisteredOrJustConsented) {
			return true
		}

		var emailPasswordValid = (this.isEmailFieldValid() && this.isPasswordFieldValid()) || !this.props.collectEmailAndPassword
		var relationshipValid = this.isRelationshipFieldValid || !this.props.collectRelationship
		return emailPasswordValid
			&& relationshipValid
			&& this.isTermsAndPrivacyFieldValid()
			&& this.isContentToUseOfTelehealthFieldValid()
	},
	isEmailFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.email)
	},
	isPasswordFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.password)
	},
	isRelationshipFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.relationship)
	},
	isTermsAndPrivacyFieldValid: function(): bool {
		return this.state.consentedToTermsAndPrivacy
	},
	isContentToUseOfTelehealthFieldValid: function(): bool {
		return this.state.consentedToUseOfTelehealth
	},


	render: function(): any {
		var hasConsented = this.state.store.userInput.consents.consentedToTermsAndPrivacy && this.state.store.userInput.consents.consentedToConsentToUseOfTelehealth
		var justRegisteredOrJustConsented = (this.state.store.parentAccount.isSignedIn && !this.state.store.parentAccount.WasSignedInAtPageLoad) || hasConsented

		var individualAgreementContainerStyle = {
			paddingTop: "16px",
			paddingBottom: "16px",
			width: "100%",
		}
		var checkboxLabelContainerStyle = {
			paddingRight: "16",
			width: "80%",
		}
		var checkboxLabelSubtextStyle = {
			color: Constants.placeholderTextColor,
			marginTop: "4",
		}
		var checkboxOuterContainerStyle = {
			marginTop: "auto",
			marginBottom: "auto",
			marginLeft: "16",
			width: "20",
			height: "20",
		}
		var checkboxInnerContainerStyle = {
			verticalAlign: "middle",
			display: "inline-block",
			width: "20",
			height: "20",
		}

		var orangeBottomDividerStyle = {
			borderBottomColor: "#F5A623",
			borderBottomWidth: "2px",
		}

		var autoCapitalize = "sentences"
		if (IsIOS) {
			// We're turning off auto capitalization because for some reason on Mobile Safari
			autoCapitalize = "off"
		}

		var topContent
		if (this.props.collectEmailAndPassword) {
			var emailHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isEmailFieldValid() : false)
			var passwordHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isPasswordFieldValid() : false)
			topContent = (
				<div>
					<div className="formFieldRow hasBottomDivider" style={emailHighlighted ? orangeBottomDividerStyle : null}>
						<input
							disabled={justRegisteredOrJustConsented}
						 	type="email"
							mozactionhint="next"
							autoCorrect="on"
							placeholder="Email Address"
							valueLink={this.linkState('email')}
							autoCapitalize={autoCapitalize} />
					</div>
					<div className="formFieldRow hasBottomDivider" style={passwordHighlighted ? orangeBottomDividerStyle : null}>
						<input 
							disabled={justRegisteredOrJustConsented}
							type="password"
							mozactionhint="next"
							placeholder="Password"
							valueLink={this.linkState('password')}
							autoCapitalize={autoCapitalize} />
					</div>
				</div>
			)
		} else if (this.props.collectRelationship) {
			var relationshipHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isRelationshipFieldValid() : false)
			var selectContainerStyle = {
				backgroundImage: "url(" + Utils.staticURL("/img/pc/select_arrow@2x.png") + ")",
				backgroundRepeat: "no-repeat",
				backgroundSize: "12px 7px",
				backgroundPosition: "right",
			};
			var style = Utils.mergeProperties(selectContainerStyle, relationshipHighlighted ? orangeBottomDividerStyle : null)
			topContent = (
				<div className="formFieldRow hasBottomDivider hasTopDivider" style={style}>
					<select
						disabled={justRegisteredOrJustConsented}
						className={this.isRelationshipFieldValid() ? null : "emptyState"}
						defaultValue=""
						valueLink={this.linkState('relationship')}>
						<option value="">Relationship to Child</option>
						<option value="mother">Mother</option>
						<option value="father">Father</option>
						<option value="other">Other Legal Guardian</option>
					</select>
				</div>
			)
		}

		var termsAndPrivacyHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isTermsAndPrivacyFieldValid() : false)
		var consentToUseOfTelehealthHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isContentToUseOfTelehealthFieldValid() : false)

		var signInURL = "/login?email=" + this.state.email + "&next=%2Fpc%2F" + ParentalConsentHydration.ChildDetails.patientID + "%2Fconsent"

		var signInCTA = null
		if (this.state.showSignInCTA) {
			signInCTA = (
				<div style={{
					textAlign: "center",
					marginBottom: "22px",
				}}>
					<a href={signInURL} onClick={function (e: any) {
						// Warning: this is a synchronous request
						Analytics.record(AnalyticsScreenName + "_sign_in_link_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName})
					}}>Sign in to an existing Spruce account.</a>
				</div>)
		}

		return (
			<form
				autoComplete="off"
				onSubmit={this.handleSubmit}
				style={{marginTop: "13"}}>
				{signInCTA}
				{topContent}
				<div className="hasBottomDivider"
					style={{
						paddingTop: "32",
						paddingBottom: "16",
						fontFamily: "MuseoSans-500",
						color: (termsAndPrivacyHighlighted || consentToUseOfTelehealthHighlighted ? "#F5A623" : ""),
					}}>
					By checking below, you are representing that you are the parent or legal guardian of the teen who initiated this visit and you are agreeing to the documents below on behalf of yourself and your teen.
				</div>
				<div style={individualAgreementContainerStyle} className="flexBox justifyContentSpaceBetween hasBottomDivider">
					<div style={checkboxLabelContainerStyle}>
						<div>
							<a href="https://d2bln09x7zhlg8.cloudfront.net/terms" onClick={function (e: any) {
								// Warning: this is a synchronous request
								Analytics.record(AnalyticsScreenName + "_terms_link_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName}, true)
							}}>Terms & Privacy Policy</a>
						</div>
						<div style={checkboxLabelSubtextStyle}>
							<label htmlFor="termsAndPrivacyCheckbox">
								Terms of use and how Spruce protects your privacy
							</label>
						</div>
					</div>
					<div style={checkboxOuterContainerStyle}>
						<div style={checkboxInnerContainerStyle}>
							<input
								disabled={justRegisteredOrJustConsented}
								type="checkbox"
								id="termsAndPrivacyCheckbox"
								checkedLink={this.linkState('consentedToTermsAndPrivacy')}
								onClick={function(){
									Analytics.record(AnalyticsScreenName + "_terms_checkbox_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName, "new_checked_value": !this.state.consentedToTermsAndPrivacy})
								}.bind(this)}
								className={(termsAndPrivacyHighlighted ? "error" : null)} />
						</div>
					</div>
				</div>

				<div style={individualAgreementContainerStyle} className="flexBox justifyContentSpaceBetween hasBottomDivider">
					<div style={checkboxLabelContainerStyle}>
						<div>
							<a href="https://d2bln09x7zhlg8.cloudfront.net/consent" onClick={function (e: any) {
								// Warning: this is a synchronous request
								Analytics.record(AnalyticsScreenName + "_telehealth_consent_link_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName}, true)
							}}>Consent to Use of Telehealth</a>
						</div>
						<div style={checkboxLabelSubtextStyle}>
							<label htmlFor="consentToUseOfTelehealth">
								You understand the benefits and risks of remote physician treatment.
							</label>
						</div>
					</div>
					<div style={checkboxOuterContainerStyle}>
						<div style={checkboxInnerContainerStyle}>
							<input
								disabled={justRegisteredOrJustConsented}
								type="checkbox"
								id="consentToUseOfTelehealth"
								checkedLink={this.linkState('consentedToUseOfTelehealth')}
								onClick={function(){
									Analytics.record(AnalyticsScreenName + "_telehealth_consent_checkbox_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName, "new_checked_value": !this.state.consentedToUseOfTelehealth})
								}.bind(this)}
								className={(consentToUseOfTelehealthHighlighted ? "error" : null)} />
						</div>
					</div>
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

module.exports = EmailRelationshipConsentView;