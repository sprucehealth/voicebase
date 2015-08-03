/* @flow */

var React = require("react");
var Reflux = require('reflux')

var Utils = require("../../libs/utils.js");
var Constants = require("./Constants.js");

var ParentalConsentStore = require('./ParentalConsentStore.js');
var SubmitButtonView = require("./SubmitButtonView.js");

var ConfirmationView = React.createClass({displayName: "ConfirmationView",
	mixins: [
		Reflux.connect(ParentalConsentStore, 'store'),
	],
	propTypes: {
		onFormSubmit: React.PropTypes.func.isRequired,
	},
	handleSubmit: function(e: any) {
		e.preventDefault();
		this.props.onFormSubmit({})
	},
	render: function(): any {
		var firstName: string = this.state.store.childDetails.firstName
		var personalPronoun: string = this.state.store.childDetails.personalPronoun
		var possessivePronoun: string = this.state.store.childDetails.possessivePronoun
		return (
			<form 
				onSubmit={this.handleSubmit}>
				<div style={{
						marginTop: "20px",
						marginBottom: "24px",
						fontFamily: "MuseoSans-500",
						fontSize: "20px",
						lineHeight: "24px",
						textAlign: "center",
					}}>
					Thanks for helping {firstName} take care of {possessivePronoun} acne. {Utils.capitalizeFirstLetter(personalPronoun)} is now able to pay for and submit {possessivePronoun} visit.
				</div>
				<div style={{textAlign: "center"}}>
					<img src={Utils.staticURL("/img/pc/completion_check@2x.png")} style={{width: "100px", height: "100px"}}/>
				</div>
				<div style={{marginTop: "26px", textAlign: "center"}}>
					Before {this.state.store.childDetails.firstName} submits {possessivePronoun} visit to a dermatologist, you can review the information that {personalPronoun} has entered so far.
				</div>
				<div>
					<SubmitButtonView 
						title="REVIEW VISIT INFORMATION"
						appearsDisabled={false}/>
				</div>
			</form>
		);
	}
});

module.exports = ConfirmationView;