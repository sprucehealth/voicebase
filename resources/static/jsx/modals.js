/** @jsx React.DOM */

module.exports = {
	ModalForm: React.createClass({displayName: "ModalForm",
		propTypes: {
			id: React.PropTypes.string.isRequired,
			title: React.PropTypes.node.isRequired,
			cancelButtonTitle: React.PropTypes.string.isRequired,
			submitButtonTitle: React.PropTypes.string.isRequired,
			onSubmit: React.PropTypes.func.isRequired
		},
		onSubmit: function(e) {
			e.preventDefault();
			if (!this.props.onSubmit(e)) {
				$("#"+this.props.id).modal('hide');
			}
			return false;
		},
		render: function() {
			return (
				<div className="modal fade" id={this.props.id} role="dialog" tabIndex="-1">
					<div className="modal-dialog">
						<div className="modal-content">
							<form role="form" onSubmit={this.onSubmit}>
								<div className="modal-header">
									<button type="button" className="close" data-dismiss="modal"><span aria-hidden="true">&times;</span><span className="sr-only">Close</span></button>
									<h4 className="modal-title">{this.props.title}</h4>
								</div>
								<div className="modal-body">
									{this.props.children}
								</div>
								<div className="modal-footer">
									<button type="button" className="btn btn-default" data-dismiss="modal">{this.props.cancelButtonTitle}</button>
									<button type="submit" className="btn btn-primary">{this.props.submitButtonTitle}</button>
								</div>
							</form>
						</div>
					</div>
				</div>
			);
		}
	})
};
