/* @flow */

import * as React from "react/addons";
import * as Utils from "./utils"
import { signup } from "./api.js"

var ApplyForm = React.createClass({displayName: "ApplyForm",

	//
	// React
	//	
	mixins: [
		React.addons.LinkedStateMixin,
	],
	getInitialState: function() {
		return {
			isSubmitting: false,
			serverProvidedUserFacingErrorMessage: "",
			clientProvidedUserFacingErrorMessage: "",
			submitButtonPressedOnce: false,

			// Form values
			firstName: "",
			lastName: "",
			email: "",
			licensedLocations: "",
			reasonsInterested: "",
			dermatologyInterests: "",
			referralSource: "",
		}
	},

	//
	// User interaction callbacks
	//
	handleSubmit: function(e: any) {
		e.preventDefault();

		this.setState({submitButtonPressedOnce: true})

		if (!this.shouldAllowSubmit()) {
			this.setState({clientProvidedUserFacingErrorMessage: "Please fill in the required fields."})

			var element = null
			if (!this.isFirstNameFieldValid()) {
				element = React.findDOMNode(this.refs.first_name)
			} else if (!this.isLastNameFieldValid()) {
				element = React.findDOMNode(this.refs.last_name)
			} else if (!this.isEmailFieldValid()) {
				element = React.findDOMNode(this.refs.email)
			} else if (!this.isLicensedLocationsFieldValid()) {
				element = React.findDOMNode(this.refs.licensed_locations)
			} else if (!this.isReasonsInterestedFieldValid()) {
				element = React.findDOMNode(this.refs.reasons_interested)
			} else if (!this.isDermatologyInterestsFieldValid()) {
				element = React.findDOMNode(this.refs.dermatology_interests)
			} else if (!this.isReferralSourceFieldValid()) {
				element = React.findDOMNode(this.refs.referral_source)
			}

			if (element) {
				element.scrollIntoView({block: "start", behavior: "smooth"})
			}
			return
		}

		function checkStatus(response: any) {
			if (response.status >= 200 && response.status < 300) {
				return response
			} else {
				var error: any = new Error(response.statusText)
				error.response = response
				throw error
			}
		}

		this.setState({isSubmitting: true})

		var component = this

		signup(this.state.firstName, this.state.lastName, this.state.email, this.state.licensedLocations, this.state.reasonsInterested, this.state.dermatologyInterests, this.state.referralSource)
			.then(checkStatus)
			.then(function(data) {
				window.location = "/thanks"
			})
			.catch(function(error) {
				var serverProvidedUserFacingErrorMessage = "Oops! Something went wrong during submission. Please try again."
				if (error.response) {
					error.response.json()
						.then(function(json) {
							if (json.error && json.error.message) {
								component.setState({
									isSubmitting: false,
									serverProvidedUserFacingErrorMessage: json.error.message,
								})
							} else {
								throw new Error()
							}
						})
						.catch(function(error) {
							component.setState({
								isSubmitting: false,
								serverProvidedUserFacingErrorMessage: serverProvidedUserFacingErrorMessage,
							})
						})
				} else {
					component.setState({
						isSubmitting: false,
						serverProvidedUserFacingErrorMessage: serverProvidedUserFacingErrorMessage,
					})
				}
			})
	},

	//
	// Internal
	//
	shouldAllowSubmit: function(): bool {
		return !this.state.isSubmitting
			&& this.isFirstNameFieldValid()
			&& this.isLastNameFieldValid()
			&& this.isEmailFieldValid()
			&& this.isLicensedLocationsFieldValid()
			&& this.isReasonsInterestedFieldValid()
			&& this.isDermatologyInterestsFieldValid()
			&& this.isReferralSourceFieldValid()
	},
	isFirstNameFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.firstName)
	},
	isLastNameFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.lastName)
	},
	isEmailFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.email)
	},
	isLicensedLocationsFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.licensedLocations)
	},
	isReasonsInterestedFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.reasonsInterested)
	},
	isDermatologyInterestsFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.dermatologyInterests)
	},
	isReferralSourceFieldValid: function(): bool {
		return !Utils.isEmpty(this.state.referralSource)
	},
	userFacingError: function(): string {
		if (!Utils.isEmpty(this.state.clientProvidedUserFacingErrorMessage)) {
			return this.state.clientProvidedUserFacingErrorMessage
		} else if (!Utils.isEmpty(this.state.serverProvidedUserFacingErrorMessage)) {
			return this.state.serverProvidedUserFacingErrorMessage
		} else {
			return ""
		}
	},

	render: function(): any {
		var errorContainerStyle = {
			hidden: this.state.serverProvidedUserFacingErrorMessage ? true : false
		}

		var userFacingError = this.userFacingError();

		var firstNameHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isFirstNameFieldValid() : false)
		var lastNameHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isLastNameFieldValid() : false)
		var emailHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isEmailFieldValid() : false)
		var licensedLocationsHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isLicensedLocationsFieldValid() : false)
		var reasonsInterestedHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isReasonsInterestedFieldValid() : false)
		var dermatologyInterestsHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isDermatologyInterestsFieldValid() : false)
		var referralSourceHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isReferralSourceFieldValid() : false)

		return (
			<form method="post" onSubmit={this.handleSubmit}>
				<div className="form-row">
					<div className="form-column" ref="first_name">
						<div>
							First Name
						</div>
						<div className="flexy-container">
							<input 
								type="text" 
								name="first_name" 
								className={firstNameHighlighted ? "error" : null}
								valueLink={this.linkState('firstName')} />
						</div>
					</div>
					<div className="form-column right" ref="last_name">
						<div>
							Last Name
						</div>
						<div className="flexy-container">
							<input 
								type="text" 
								name="last_name" 
								className={lastNameHighlighted ? "error" : null}
								valueLink={this.linkState('lastName')} />
						</div>
					</div>
				</div>
				<div className="form-row">
					<div className="form-column" ref="email">
						<div>
							Email
						</div>
						<div className="flexy-container">
							<input 
								type="email" 
								name="email" 
								className={emailHighlighted ? "error" : null}
								valueLink={this.linkState('email')} />
						</div>
					</div>
				</div>
				<div className="form-row">
					<div className="form-column" ref="licensed_locations">
						<div>
							Where are you licensed?
						</div>
						<div className="flexy-container">
							<input 
								type="text" 
								name="licensed_locations" 
								className={licensedLocationsHighlighted ? "error" : null}
								valueLink={this.linkState('licensedLocations')} />
						</div>
					</div>
				</div>
				<div className="form-row">
					<div className="form-column" ref="reasons_interested" >
						<div>
							Why are you interested in joining?
						</div>
						<div className="flexy-container">
							<textarea 
								name="reasons_interested" 
								className={reasonsInterestedHighlighted ? "error" : null}
								valueLink={this.linkState('reasonsInterested')}>
							</textarea>
						</div>
					</div>
				</div>
				<div className="form-row">
					<div className="form-column" ref="dermatology_interests">
						<div>
							What are your interest areas within dermatology?
						</div>
						<div className="flexy-container">
							<textarea 
								name="dermatology_interests"  
								className={dermatologyInterestsHighlighted ? "error" : null}
								valueLink={this.linkState('dermatologyInterests')}>
							</textarea>
						</div>
					</div>
				</div>
				<div className="form-row">
					<div className="form-column" ref="referral_source">
						<div>
							How did you hear about us?
						</div>
						<div className="flexy-container">
							<input 
								type="text" 
								name="referral_source" 
								className={referralSourceHighlighted ? "error" : null}
								valueLink={this.linkState('referralSource')} />
						</div>
					</div>
				</div>
				<div style={errorContainerStyle} className={!Utils.isEmpty(userFacingError) ? "form-error-message" : null}>
					{userFacingError}
				</div>
				<div className="flexy-container">
					<button type="submit" disabled={this.isSubmitting}>Submit Application</button>
				</div>
			</form>
		);
	}
});

React.render(
	<ApplyForm />,
	document.getElementById('apply-form')
);