var _tapglue$snaas$Native_LocalStorage = function() {
	// Polyfill storage implementation so we can test the behaviour in isolation.
	if (!('localStorage' in window)) {
		window.localStorage = {
			_data       : {},
			setItem     : function(id, val) { return this._data[id] = String(val); },
			getItem     : function(id) { return this._data.hasOwnProperty(id) ? this._data[id] : null; },
			removeItem  : function(id) { return delete this._data[id]; },
			clear       : function() { return this._data = {}; }
		};
	}

	function disabled() {
		return _elm_lang$core$Result$Err({ ctor: 'Disabled' });
	}

	function get(key) {
		var val = window.localStorage.getItem(key);

		if (val === null) {
			return _elm_lang$core$Result$Err({ ctor: 'NotFound' });
		}

		return _elm_lang$core$Result$Ok(val);
	}

	function keys() {
		var keyList = _elm_lang$core$Native_List.Nil;

		for (var i = window.localStorage.length; i--;) {
			keyList = _elm_lang$core$Native_List.Cons(window.localStorage.key(i), keyList);
		}

		return _elm_lang$core$Result$Ok(keyList);
	}

	function remove(key) {
		window.localStorage.removeItem(key);

		return _elm_lang$core$Result$Ok(_elm_lang$core$Native_Utils.Tuple0);
	}

	function set(key, val) {
		try {
			window.localStorage.setItem(key, val);

			return _elm_lang$core$Result$Ok(_elm_lang$core$Native_Utils.Tuple0);
		} catch (err) {
			return _elm_lang$core$Result$Err({ ctor: 'QuotaExceeded' });
		}
	}

	return {
		clear: disabled(),
		get: get,
		keys: keys,
		remove: remove,
		set: F2(set)
	};
}();