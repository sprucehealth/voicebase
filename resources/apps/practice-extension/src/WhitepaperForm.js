/* @flow */

import * as React from "react/addons"
import { isEmpty } from "../../libs/emptiness.js"
import { submitWhitepaperRequest } from "./api.js" 

var WhitepaperForm = React.createClass({displayName: "WhitepaperForm",

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

		submitWhitepaperRequest(this.state.firstName, this.state.lastName, this.state.email)
			.then(checkStatus)
			.then(function(data) {
		        window.location = "https://d2bln09x7zhlg8.cloudfront.net/whitepaper.pdf";
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
	},
	isFirstNameFieldValid: function(): bool {
		return !isEmpty(this.state.firstName)
	},
	isLastNameFieldValid: function(): bool {
		return !isEmpty(this.state.lastName)
	},
	isEmailFieldValid: function(): bool {
		return !isEmpty(this.state.email)
	},
	userFacingError: function(): string {
		if (!isEmpty(this.state.clientProvidedUserFacingErrorMessage)) {
			return this.state.clientProvidedUserFacingErrorMessage
		} else if (!isEmpty(this.state.serverProvidedUserFacingErrorMessage)) {
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
				
				<div style={errorContainerStyle} className={!isEmpty(userFacingError) ? "form-error-message" : null}>
					{userFacingError}
				</div>
				<div className="flexy-container">
					<button type="submit" disabled={this.isSubmitting}>Download Whitepaper</button>
				</div>
			</form>
		);
	}
});

React.render(
	<WhitepaperForm />,
	document.getElementById('form-container')
);