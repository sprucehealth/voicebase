/* @flow */

import * as React from "react/addons"
import * as Emptiness from "../../libs/emptiness.js" 
import { submitDemoRequest } from "./api.js" 

var RequestDemoForm = React.createClass({displayName: "ApplyForm",

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
			submitButtonPressedOnce: false,

			// Form values
			firstName: "",
			lastName: "",
			email: "",
			phone: "",
			state: "",
		}
	},

	//
	// User interaction callbacks
	//
	handleSubmit: function(e: any) {
		e.preventDefault();

		this.setState({submitButtonPressedOnce: true})

		if (!this.shouldAllowSubmit()) {

			var element = null
			if (!this.isFirstNameFieldValid()) {
				element = React.findDOMNode(this.refs.first_name)
			} else if (!this.isLastNameFieldValid()) {
				element = React.findDOMNode(this.refs.last_name)
			} else if (!this.isEmailFieldValid()) {
				element = React.findDOMNode(this.refs.email)
			} else if (!this.isPhoneFieldValid()) {
				element = React.findDOMNode(this.refs.state)
			} else if (!this.isStateFieldValid()) {
				element = React.findDOMNode(this.refs.reasons_interested)
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

		submitDemoRequest(this.state.firstName, this.state.lastName, this.state.email, this.state.phone, this.state.state)
			.then(checkStatus)
			.then(function(data) {
				window.location = "/practices/thanks"
			})
			.catch(function(error) {
				var serverProvidedUserFacingErrorMessage = "Oops! Our server reported an error during submission. Please check all fields and try again, or email us at support@sprucehealth.com."
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
			&& this.isPhoneFieldValid()
			&& this.isStateFieldValid()
	},
	isFirstNameFieldValid: function(): bool {
		return !Emptiness.isEmpty(this.state.firstName)
	},
	isLastNameFieldValid: function(): bool {
		return !Emptiness.isEmpty(this.state.lastName)
	},
	isEmailFieldValid: function(): bool {
		return !Emptiness.isEmpty(this.state.email)
	},
	isPhoneFieldValid: function(): bool {
		return !Emptiness.isEmpty(this.state.phone)
	},
	isStateFieldValid: function(): bool {
		return !Emptiness.isEmpty(this.state.state)
	},
	userFacingError: function(): string {
		if (this.state.submitButtonPressedOnce && !this.shouldAllowSubmit() && !this.state.isSubmitting) {
			return "Please fill in the required fields."
		} else if (!Emptiness.isEmpty(this.state.serverProvidedUserFacingErrorMessage)) {
			return this.state.serverProvidedUserFacingErrorMessage
		} else {
			return ""
		}
	},

	render: function(): any {

		var userFacingError = this.userFacingError();

		var firstNameHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isFirstNameFieldValid() : false)
		var lastNameHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isLastNameFieldValid() : false)
		var emailHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isEmailFieldValid() : false)
		var phoneHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isPhoneFieldValid() : false)
		var stateHighlighted: bool = (this.state.submitButtonPressedOnce ? !this.isStateFieldValid() : false)

		return (
			<form method="post" onSubmit={this.handleSubmit}>
				<h3 id="get-started">Get Started</h3>
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
				</div>
				<div className="form-row">
					<div className="form-column" ref="last_name">
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
					<div className="form-column" ref="phone">
						<div>
							Phone Number
						</div>
						<div className="flexy-container">
							<input 
								type="tel" 
								name="phone" 
								className={phoneHighlighted ? "error" : null}
								valueLink={this.linkState('phone')} />
						</div>
					</div>
				</div>
				<div className="form-row">
					<div className="form-column" ref="state">
						<div>
							State
						</div>
						<div className="flexy-container">
							<input 
								type="text" 
								name="state" 
								className={stateHighlighted ? "error" : null}
								valueLink={this.linkState('state')} />
						</div>
					</div>
				</div>
				<div className="form-error-message">
					{Emptiness.isEmpty(userFacingError) ? (null) : userFacingError}
				</div>
				<div className="flexy-container">
					<button type="submit" disabled={this.isSubmitting}>Request a Demo</button>
				</div>
			</form>
		);
	}
});

React.render(
	<RequestDemoForm />,
	document.getElementById('apply-form-container')
);