var _tapglue$snaas$Native_Confirm = function() {
	function dialog(question) {
		return _elm_lang$core$Result$Ok(window.confirm(question));
	}

	return {
		dialog: dialog
	};
}();