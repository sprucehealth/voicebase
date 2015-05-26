/* @flow */

var React = require("react");

module.exports = {
	FormSelect: React.createClass({displayName: "FormSelect",
		propTypes: {
			name: React.PropTypes.string,
			label: React.PropTypes.string,
			value: React.PropTypes.oneOfType([
				React.PropTypes.string,
				React.PropTypes.number
			]),
			opts: React.PropTypes.arrayOf(React.PropTypes.shape({
				name: React.PropTypes.string.isRequired,
				value: React.PropTypes.oneOfType([
					React.PropTypes.string,
					React.PropTypes.number
				]).isRequired,
			})).isRequired,
			onChange: React.PropTypes.func
		},
		getDefaultProps: function(): any {
			return {opts: []};
		},
		render: function(): any {
			return (
				<div className="form-group">
					{this.props.label ? <span><label className="control-label" htmlFor={this.props.name}>{this.props.label}</label><br /></span> : null}
					<select name={this.props.name} className="form-control" value={this.props.value} onChange={this.props.onChange}>
						{this.props.opts.map(function(opt) {
							return <option key={"select-value-" + opt.value} value={opt.value}>{opt.name}</option>
						}.bind(this))}
					</select>
				</div>
			);
		}
	}),

	FormInput: React.createClass({displayName: "FormInput",
		propTypes: {
			type: React.PropTypes.string,
			name: React.PropTypes.string,
			label: React.PropTypes.node,
			value: React.PropTypes.oneOfType([
				React.PropTypes.string,
				React.PropTypes.number
			]),
			placeholder: React.PropTypes.string,
			required: React.PropTypes.bool,
			onChange: React.PropTypes.func,
			onKeyDown: React.PropTypes.func
		},
		getDefaultProps: function(): any {
			return {
				type: "text",
				required: false
			}
		},
		render: function(): any {
			return (
				<div className="form-group">
					{this.props.label ? <label className="control-label" htmlFor={this.props.name}>{this.props.label}</label> : null}
					<input required={this.props.required ? "true" : null} type={this.props.type} className="form-control section-name"
						placeholder={this.props.placeholder} name={this.props.name} value={this.props.value}
						onKeyDown={this.props.onKeyDown} onChange={this.props.onChange} />
				</div>
			);
		}
	}),

	Checkbox: React.createClass({displayName: "Checkbox",
		propTypes: {
			name: React.PropTypes.string,
			label: React.PropTypes.node,
			checked: React.PropTypes.bool,
			onChange: React.PropTypes.func.isRequired,
		},
		onChange: function(e: any): void {
			e.preventDefault();
			this.props.onChange(e, e.target.checked);
		},
		render: function(): any {
			// FIXME: Avert your eyes for below is a hack to get around the checkbox not working if only the checked
			// values changes. It's madness. I'm guessing reactjs bug but need to prove it.
			return (
				<div>
					{this.props.checked ?
						<span><input name={this.props.name} checked="true" onChange={this.onChange} type="checkbox" /></span>
					:
						<input name={this.props.name} checked="" onChange={this.onChange} type="checkbox" />
					}
					{this.props.label ? <strong> {this.props.label}</strong> : null}
				</div>
			);
		}
	}),

	TextArea: React.createClass({displayName: "TextArea",
		propTypes: {
			name: React.PropTypes.string,
			label: React.PropTypes.node,
			value: React.PropTypes.string,
			rows: React.PropTypes.number,
			tabs: React.PropTypes.bool,
			onChange: React.PropTypes.func,
		},
		getDefaultProps: function(): any {
			return {
				rows: 5,
				tabs: false
			}
		},
		onKeyDown: function(e: any): void {
			if (!this.props.tabs) {
				return;
			}
			var keyCode = e.keyCode || e.which;
			if (keyCode == 9) {
				e.preventDefault();
				var start = $(e.target).get(0).selectionStart;
				var end = $(e.target).get(0).selectionEnd;
				$(e.target).val($(e.target).val().substring(0, start) + "\t" + $(e.target).val().substring(end));
				$(e.target).get(0).selectionStart =
				$(e.target).get(0).selectionEnd = start + 1;
			  }
		},
		render: function(): any {
			return (
				<div className="form-group">
					{this.props.label ? <label className="control-label" htmlFor={this.props.name}>{this.props.label}</label> : null}
					<textarea type="text" className="form-control section-name" rows={this.props.rows}
						   name={this.props.name} value={this.props.value} onChange={this.props.onChange}
						   onKeyDown={this.onKeyDown} />
				</div>
			);
		}
	})
};
