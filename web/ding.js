"use strict";
// Javascript is generated from typescript, do not modify generated javascript because changes will be overwritten.
const [dom, style, attr, prop] = (function () {
	// Start of unicode block (rough approximation of script), from https://www.unicode.org/Public/UNIDATA/Blocks.txt
	const scriptblocks = [0x0000, 0x0080, 0x0100, 0x0180, 0x0250, 0x02B0, 0x0300, 0x0370, 0x0400, 0x0500, 0x0530, 0x0590, 0x0600, 0x0700, 0x0750, 0x0780, 0x07C0, 0x0800, 0x0840, 0x0860, 0x0870, 0x08A0, 0x0900, 0x0980, 0x0A00, 0x0A80, 0x0B00, 0x0B80, 0x0C00, 0x0C80, 0x0D00, 0x0D80, 0x0E00, 0x0E80, 0x0F00, 0x1000, 0x10A0, 0x1100, 0x1200, 0x1380, 0x13A0, 0x1400, 0x1680, 0x16A0, 0x1700, 0x1720, 0x1740, 0x1760, 0x1780, 0x1800, 0x18B0, 0x1900, 0x1950, 0x1980, 0x19E0, 0x1A00, 0x1A20, 0x1AB0, 0x1B00, 0x1B80, 0x1BC0, 0x1C00, 0x1C50, 0x1C80, 0x1C90, 0x1CC0, 0x1CD0, 0x1D00, 0x1D80, 0x1DC0, 0x1E00, 0x1F00, 0x2000, 0x2070, 0x20A0, 0x20D0, 0x2100, 0x2150, 0x2190, 0x2200, 0x2300, 0x2400, 0x2440, 0x2460, 0x2500, 0x2580, 0x25A0, 0x2600, 0x2700, 0x27C0, 0x27F0, 0x2800, 0x2900, 0x2980, 0x2A00, 0x2B00, 0x2C00, 0x2C60, 0x2C80, 0x2D00, 0x2D30, 0x2D80, 0x2DE0, 0x2E00, 0x2E80, 0x2F00, 0x2FF0, 0x3000, 0x3040, 0x30A0, 0x3100, 0x3130, 0x3190, 0x31A0, 0x31C0, 0x31F0, 0x3200, 0x3300, 0x3400, 0x4DC0, 0x4E00, 0xA000, 0xA490, 0xA4D0, 0xA500, 0xA640, 0xA6A0, 0xA700, 0xA720, 0xA800, 0xA830, 0xA840, 0xA880, 0xA8E0, 0xA900, 0xA930, 0xA960, 0xA980, 0xA9E0, 0xAA00, 0xAA60, 0xAA80, 0xAAE0, 0xAB00, 0xAB30, 0xAB70, 0xABC0, 0xAC00, 0xD7B0, 0xD800, 0xDB80, 0xDC00, 0xE000, 0xF900, 0xFB00, 0xFB50, 0xFE00, 0xFE10, 0xFE20, 0xFE30, 0xFE50, 0xFE70, 0xFF00, 0xFFF0, 0x10000, 0x10080, 0x10100, 0x10140, 0x10190, 0x101D0, 0x10280, 0x102A0, 0x102E0, 0x10300, 0x10330, 0x10350, 0x10380, 0x103A0, 0x10400, 0x10450, 0x10480, 0x104B0, 0x10500, 0x10530, 0x10570, 0x10600, 0x10780, 0x10800, 0x10840, 0x10860, 0x10880, 0x108E0, 0x10900, 0x10920, 0x10980, 0x109A0, 0x10A00, 0x10A60, 0x10A80, 0x10AC0, 0x10B00, 0x10B40, 0x10B60, 0x10B80, 0x10C00, 0x10C80, 0x10D00, 0x10E60, 0x10E80, 0x10EC0, 0x10F00, 0x10F30, 0x10F70, 0x10FB0, 0x10FE0, 0x11000, 0x11080, 0x110D0, 0x11100, 0x11150, 0x11180, 0x111E0, 0x11200, 0x11280, 0x112B0, 0x11300, 0x11400, 0x11480, 0x11580, 0x11600, 0x11660, 0x11680, 0x11700, 0x11800, 0x118A0, 0x11900, 0x119A0, 0x11A00, 0x11A50, 0x11AB0, 0x11AC0, 0x11B00, 0x11C00, 0x11C70, 0x11D00, 0x11D60, 0x11EE0, 0x11F00, 0x11FB0, 0x11FC0, 0x12000, 0x12400, 0x12480, 0x12F90, 0x13000, 0x13430, 0x14400, 0x16800, 0x16A40, 0x16A70, 0x16AD0, 0x16B00, 0x16E40, 0x16F00, 0x16FE0, 0x17000, 0x18800, 0x18B00, 0x18D00, 0x1AFF0, 0x1B000, 0x1B100, 0x1B130, 0x1B170, 0x1BC00, 0x1BCA0, 0x1CF00, 0x1D000, 0x1D100, 0x1D200, 0x1D2C0, 0x1D2E0, 0x1D300, 0x1D360, 0x1D400, 0x1D800, 0x1DF00, 0x1E000, 0x1E030, 0x1E100, 0x1E290, 0x1E2C0, 0x1E4D0, 0x1E7E0, 0x1E800, 0x1E900, 0x1EC70, 0x1ED00, 0x1EE00, 0x1F000, 0x1F030, 0x1F0A0, 0x1F100, 0x1F200, 0x1F300, 0x1F600, 0x1F650, 0x1F680, 0x1F700, 0x1F780, 0x1F800, 0x1F900, 0x1FA00, 0x1FA70, 0x1FB00, 0x20000, 0x2A700, 0x2B740, 0x2B820, 0x2CEB0, 0x2F800, 0x30000, 0x31350, 0xE0000, 0xE0100, 0xF0000, 0x100000];
	// Find block code belongs in.
	const findBlock = (code) => {
		let s = 0;
		let e = scriptblocks.length;
		while (s < e - 1) {
			let i = Math.floor((s + e) / 2);
			if (code < scriptblocks[i]) {
				e = i;
			}
			else {
				s = i;
			}
		}
		return s;
	};
	// formatText adds s to element e, in a way that makes switching unicode scripts
	// clear, with alternating DOM TextNode and span elements with a "switchscript"
	// class. Useful for highlighting look alikes, e.g. a (ascii 0x61) and а (cyrillic
	// 0x430).
	//
	// This is only called one string at a time, so the UI can still display strings
	// without highlighting switching scripts, by calling formatText on the parts.
	const formatText = (e, s) => {
		// Handle some common cases quickly.
		if (!s) {
			return;
		}
		let ascii = true;
		for (const c of s) {
			const cp = c.codePointAt(0); // For typescript, to check for undefined.
			if (cp !== undefined && cp >= 0x0080) {
				ascii = false;
				break;
			}
		}
		if (ascii) {
			e.appendChild(document.createTextNode(s));
			return;
		}
		// todo: handle grapheme clusters? wait for Intl.Segmenter?
		let n = 0; // Number of text/span parts added.
		let str = ''; // Collected so far.
		let block = -1; // Previous block/script.
		let mod = 1;
		const put = (nextblock) => {
			if (n === 0 && nextblock === 0) {
				// Start was non-ascii, second block is ascii, we'll start marked as switched.
				mod = 0;
			}
			if (n % 2 === mod) {
				const x = document.createElement('span');
				x.classList.add('scriptswitch');
				x.appendChild(document.createTextNode(str));
				e.appendChild(x);
			}
			else {
				e.appendChild(document.createTextNode(str));
			}
			n++;
			str = '';
		};
		for (const c of s) {
			// Basic whitespace does not switch blocks. Will probably need to extend with more
			// punctuation in the future. Possibly for digits too. But perhaps not in all
			// scripts.
			if (c === ' ' || c === '\t' || c === '\r' || c === '\n') {
				str += c;
				continue;
			}
			const code = c.codePointAt(0);
			if (block < 0 || !(code >= scriptblocks[block] && (code < scriptblocks[block + 1] || block === scriptblocks.length - 1))) {
				const nextblock = code < 0x0080 ? 0 : findBlock(code);
				if (block >= 0) {
					put(nextblock);
				}
				block = nextblock;
			}
			str += c;
		}
		put(-1);
	};
	const _domKids = (e, l) => {
		l.forEach((c) => {
			const xc = c;
			if (typeof c === 'string') {
				formatText(e, c);
			}
			else if (c instanceof String) {
				// String is an escape-hatch for text that should not be formatted with
				// unicode-block-change-highlighting, e.g. for textarea values.
				e.appendChild(document.createTextNode('' + c));
			}
			else if (c instanceof Element) {
				e.appendChild(c);
			}
			else if (c instanceof Function) {
				if (!c.name) {
					throw new Error('function without name');
				}
				e.addEventListener(c.name, c);
			}
			else if (Array.isArray(xc)) {
				_domKids(e, c);
			}
			else if (xc._class) {
				for (const s of xc._class) {
					e.classList.toggle(s, true);
				}
			}
			else if (xc._attrs) {
				for (const k in xc._attrs) {
					e.setAttribute(k, xc._attrs[k]);
				}
			}
			else if (xc._styles) {
				for (const k in xc._styles) {
					const estyle = e.style;
					estyle[k] = xc._styles[k];
				}
			}
			else if (xc._props) {
				for (const k in xc._props) {
					const eprops = e;
					eprops[k] = xc._props[k];
				}
			}
			else if (xc.root) {
				e.appendChild(xc.root);
			}
			else {
				console.log('bad kid', c);
				throw new Error('bad kid');
			}
		});
		return e;
	};
	const dom = {
		_kids: function (e, ...kl) {
			while (e.firstChild) {
				e.removeChild(e.firstChild);
			}
			_domKids(e, kl);
		},
		_attrs: (x) => { return { _attrs: x }; },
		_class: (...x) => { return { _class: x }; },
		// The createElement calls are spelled out so typescript can derive function
		// signatures with a specific HTML*Element return type.
		div: (...l) => _domKids(document.createElement('div'), l),
		span: (...l) => _domKids(document.createElement('span'), l),
		a: (...l) => _domKids(document.createElement('a'), l),
		input: (...l) => _domKids(document.createElement('input'), l),
		textarea: (...l) => _domKids(document.createElement('textarea'), l),
		select: (...l) => _domKids(document.createElement('select'), l),
		option: (...l) => _domKids(document.createElement('option'), l),
		clickbutton: (...l) => _domKids(document.createElement('button'), [attr.type('button'), ...l]),
		submitbutton: (...l) => _domKids(document.createElement('button'), [attr.type('submit'), ...l]),
		form: (...l) => _domKids(document.createElement('form'), l),
		fieldset: (...l) => _domKids(document.createElement('fieldset'), l),
		table: (...l) => _domKids(document.createElement('table'), l),
		thead: (...l) => _domKids(document.createElement('thead'), l),
		tbody: (...l) => _domKids(document.createElement('tbody'), l),
		tfoot: (...l) => _domKids(document.createElement('tfoot'), l),
		tr: (...l) => _domKids(document.createElement('tr'), l),
		td: (...l) => _domKids(document.createElement('td'), l),
		th: (...l) => _domKids(document.createElement('th'), l),
		datalist: (...l) => _domKids(document.createElement('datalist'), l),
		h1: (...l) => _domKids(document.createElement('h1'), l),
		h2: (...l) => _domKids(document.createElement('h2'), l),
		h3: (...l) => _domKids(document.createElement('h3'), l),
		br: (...l) => _domKids(document.createElement('br'), l),
		hr: (...l) => _domKids(document.createElement('hr'), l),
		pre: (...l) => _domKids(document.createElement('pre'), l),
		label: (...l) => _domKids(document.createElement('label'), l),
		ul: (...l) => _domKids(document.createElement('ul'), l),
		li: (...l) => _domKids(document.createElement('li'), l),
		iframe: (...l) => _domKids(document.createElement('iframe'), l),
		b: (...l) => _domKids(document.createElement('b'), l),
		img: (...l) => _domKids(document.createElement('img'), l),
		style: (...l) => _domKids(document.createElement('style'), l),
		search: (...l) => _domKids(document.createElement('search'), l),
		p: (...l) => _domKids(document.createElement('p'), l),
		tt: (...l) => _domKids(document.createElement('tt'), l),
		i: (...l) => _domKids(document.createElement('i'), l),
		link: (...l) => _domKids(document.createElement('link'), l),
	};
	const _attr = (k, v) => { const o = {}; o[k] = v; return { _attrs: o }; };
	const attr = {
		title: (s) => _attr('title', s),
		value: (s) => _attr('value', s),
		type: (s) => _attr('type', s),
		tabindex: (s) => _attr('tabindex', s),
		src: (s) => _attr('src', s),
		placeholder: (s) => _attr('placeholder', s),
		href: (s) => _attr('href', s),
		checked: (s) => _attr('checked', s),
		selected: (s) => _attr('selected', s),
		id: (s) => _attr('id', s),
		datalist: (s) => _attr('datalist', s),
		rows: (s) => _attr('rows', s),
		target: (s) => _attr('target', s),
		rel: (s) => _attr('rel', s),
		required: (s) => _attr('required', s),
		multiple: (s) => _attr('multiple', s),
		download: (s) => _attr('download', s),
		disabled: (s) => _attr('disabled', s),
		draggable: (s) => _attr('draggable', s),
		rowspan: (s) => _attr('rowspan', s),
		colspan: (s) => _attr('colspan', s),
		for: (s) => _attr('for', s),
		role: (s) => _attr('role', s),
		arialabel: (s) => _attr('aria-label', s),
		arialive: (s) => _attr('aria-live', s),
		name: (s) => _attr('name', s),
		min: (s) => _attr('min', s),
		max: (s) => _attr('max', s),
		action: (s) => _attr('action', s),
		method: (s) => _attr('method', s),
		autocomplete: (s) => _attr('autocomplete', s),
		list: (s) => _attr('list', s),
		form: (s) => _attr('form', s),
		size: (s) => _attr('size', s),
	};
	const style = (x) => { return { _styles: x }; };
	const prop = (x) => { return { _props: x }; };
	return [dom, style, attr, prop];
})();
// NOTE: GENERATED by github.com/mjl-/sherpats, DO NOT MODIFY
var api;
(function (api) {
	// BuildStatus indicates the progress of a build.
	let BuildStatus;
	(function (BuildStatus) {
		BuildStatus["StatusNew"] = "new";
		BuildStatus["StatusClone"] = "clone";
		BuildStatus["StatusBuild"] = "build";
		BuildStatus["StatusSuccess"] = "success";
		BuildStatus["StatusCancelled"] = "cancelled";
	})(BuildStatus = api.BuildStatus || (api.BuildStatus = {}));
	// VCS indicates the mechanism to fetch the source code.
	let VCS;
	(function (VCS) {
		VCS["VCSGit"] = "git";
		VCS["VCSMercurial"] = "mercurial";
		// Custom shell script that will do the cloning. Escape hatch mechanism to support
		// past/future systems.
		VCS["VCSCommand"] = "command";
	})(VCS = api.VCS || (api.VCS = {}));
	// LogLevel indicates the severity of a log message.
	let LogLevel;
	(function (LogLevel) {
		LogLevel["LogDebug"] = "debug";
		LogLevel["LogInfo"] = "info";
		LogLevel["LogWarn"] = "warn";
		LogLevel["LogError"] = "error";
	})(LogLevel = api.LogLevel || (api.LogLevel = {}));
	api.structTypes = { "Build": true, "EventBuild": true, "EventOutput": true, "EventRemoveBuild": true, "EventRemoveRepo": true, "EventRepo": true, "GoToolchains": true, "Repo": true, "RepoBuilds": true, "Result": true, "Settings": true, "Step": true };
	api.stringsTypes = { "BuildStatus": true, "LogLevel": true, "VCS": true };
	api.intsTypes = {};
	api.types = {
		"Build": { "Name": "Build", "Docs": "", "Fields": [{ "Name": "ID", "Docs": "", "Typewords": ["int32"] }, { "Name": "RepoName", "Docs": "", "Typewords": ["string"] }, { "Name": "Branch", "Docs": "", "Typewords": ["string"] }, { "Name": "CommitHash", "Docs": "", "Typewords": ["string"] }, { "Name": "Status", "Docs": "", "Typewords": ["BuildStatus"] }, { "Name": "Created", "Docs": "", "Typewords": ["timestamp"] }, { "Name": "Start", "Docs": "", "Typewords": ["nullable", "timestamp"] }, { "Name": "Finish", "Docs": "", "Typewords": ["nullable", "timestamp"] }, { "Name": "ErrorMessage", "Docs": "", "Typewords": ["string"] }, { "Name": "Released", "Docs": "", "Typewords": ["nullable", "timestamp"] }, { "Name": "BuilddirRemoved", "Docs": "", "Typewords": ["bool"] }, { "Name": "Coverage", "Docs": "", "Typewords": ["nullable", "float32"] }, { "Name": "CoverageReportFile", "Docs": "", "Typewords": ["string"] }, { "Name": "Version", "Docs": "", "Typewords": ["string"] }, { "Name": "BuildScript", "Docs": "", "Typewords": ["string"] }, { "Name": "LowPrio", "Docs": "", "Typewords": ["bool"] }, { "Name": "LastLine", "Docs": "", "Typewords": ["string"] }, { "Name": "DiskUsage", "Docs": "", "Typewords": ["int64"] }, { "Name": "HomeDiskUsageDelta", "Docs": "", "Typewords": ["int64"] }, { "Name": "Results", "Docs": "", "Typewords": ["[]", "Result"] }, { "Name": "Steps", "Docs": "", "Typewords": ["[]", "Step"] }] },
		"Result": { "Name": "Result", "Docs": "", "Fields": [{ "Name": "Command", "Docs": "", "Typewords": ["string"] }, { "Name": "Os", "Docs": "", "Typewords": ["string"] }, { "Name": "Arch", "Docs": "", "Typewords": ["string"] }, { "Name": "Toolchain", "Docs": "", "Typewords": ["string"] }, { "Name": "Filename", "Docs": "", "Typewords": ["string"] }, { "Name": "Filesize", "Docs": "", "Typewords": ["int64"] }] },
		"Step": { "Name": "Step", "Docs": "", "Fields": [{ "Name": "Name", "Docs": "", "Typewords": ["string"] }, { "Name": "Output", "Docs": "", "Typewords": ["string"] }, { "Name": "Nsec", "Docs": "", "Typewords": ["int64"] }] },
		"RepoBuilds": { "Name": "RepoBuilds", "Docs": "", "Fields": [{ "Name": "Repo", "Docs": "", "Typewords": ["Repo"] }, { "Name": "Builds", "Docs": "", "Typewords": ["[]", "Build"] }] },
		"Repo": { "Name": "Repo", "Docs": "", "Fields": [{ "Name": "Name", "Docs": "", "Typewords": ["string"] }, { "Name": "VCS", "Docs": "", "Typewords": ["VCS"] }, { "Name": "Origin", "Docs": "", "Typewords": ["string"] }, { "Name": "DefaultBranch", "Docs": "", "Typewords": ["string"] }, { "Name": "CheckoutPath", "Docs": "", "Typewords": ["string"] }, { "Name": "BuildScript", "Docs": "", "Typewords": ["string"] }, { "Name": "UID", "Docs": "", "Typewords": ["nullable", "uint32"] }, { "Name": "HomeDiskUsage", "Docs": "", "Typewords": ["int64"] }, { "Name": "WebhookSecret", "Docs": "", "Typewords": ["string"] }, { "Name": "AllowGlobalWebhookSecrets", "Docs": "", "Typewords": ["bool"] }, { "Name": "GoAuto", "Docs": "", "Typewords": ["bool"] }, { "Name": "GoCur", "Docs": "", "Typewords": ["bool"] }, { "Name": "GoPrev", "Docs": "", "Typewords": ["bool"] }, { "Name": "GoNext", "Docs": "", "Typewords": ["bool"] }, { "Name": "Bubblewrap", "Docs": "", "Typewords": ["bool"] }, { "Name": "BubblewrapNoNet", "Docs": "", "Typewords": ["bool"] }, { "Name": "NotifyEmailAddrs", "Docs": "", "Typewords": ["[]", "string"] }, { "Name": "BuildOnUpdatedToolchain", "Docs": "", "Typewords": ["bool"] }] },
		"GoToolchains": { "Name": "GoToolchains", "Docs": "", "Fields": [{ "Name": "Go", "Docs": "", "Typewords": ["string"] }, { "Name": "GoPrev", "Docs": "", "Typewords": ["string"] }, { "Name": "GoNext", "Docs": "", "Typewords": ["string"] }] },
		"Settings": { "Name": "Settings", "Docs": "", "Fields": [{ "Name": "ID", "Docs": "", "Typewords": ["int32"] }, { "Name": "NotifyEmailAddrs", "Docs": "", "Typewords": ["[]", "string"] }, { "Name": "GithubWebhookSecret", "Docs": "", "Typewords": ["string"] }, { "Name": "GiteaWebhookSecret", "Docs": "", "Typewords": ["string"] }, { "Name": "BitbucketWebhookSecret", "Docs": "", "Typewords": ["string"] }, { "Name": "RunPrefix", "Docs": "", "Typewords": ["[]", "string"] }, { "Name": "Environment", "Docs": "", "Typewords": ["[]", "string"] }, { "Name": "AutomaticGoToolchains", "Docs": "", "Typewords": ["bool"] }] },
		"BuildStatus": { "Name": "BuildStatus", "Docs": "", "Values": [{ "Name": "StatusNew", "Value": "new", "Docs": "" }, { "Name": "StatusClone", "Value": "clone", "Docs": "" }, { "Name": "StatusBuild", "Value": "build", "Docs": "" }, { "Name": "StatusSuccess", "Value": "success", "Docs": "" }, { "Name": "StatusCancelled", "Value": "cancelled", "Docs": "" }] },
		"VCS": { "Name": "VCS", "Docs": "", "Values": [{ "Name": "VCSGit", "Value": "git", "Docs": "" }, { "Name": "VCSMercurial", "Value": "mercurial", "Docs": "" }, { "Name": "VCSCommand", "Value": "command", "Docs": "" }] },
		"LogLevel": { "Name": "LogLevel", "Docs": "", "Values": [{ "Name": "LogDebug", "Value": "debug", "Docs": "" }, { "Name": "LogInfo", "Value": "info", "Docs": "" }, { "Name": "LogWarn", "Value": "warn", "Docs": "" }, { "Name": "LogError", "Value": "error", "Docs": "" }] },
		"EventRepo": { "Name": "EventRepo", "Docs": "EventRepo represents an update of a repository or creation of a repository.", "Fields": [{ "Name": "Repo", "Docs": "", "Typewords": ["Repo"] }] },
		"EventRemoveRepo": { "Name": "EventRemoveRepo", "Docs": "EventRemoveRepo represents the removal of a repository.", "Fields": [{ "Name": "RepoName", "Docs": "", "Typewords": ["string"] }] },
		"EventBuild": { "Name": "EventBuild", "Docs": "EventBuild represents an update to a build, or the start of a new build.\nOutput is not part of the build, see EventOutput below.", "Fields": [{ "Name": "Build", "Docs": "", "Typewords": ["Build"] }] },
		"EventRemoveBuild": { "Name": "EventRemoveBuild", "Docs": "EventRemoveBuild represents the removal of a build from the database.", "Fields": [{ "Name": "RepoName", "Docs": "", "Typewords": ["string"] }, { "Name": "BuildID", "Docs": "", "Typewords": ["int32"] }] },
		"EventOutput": { "Name": "EventOutput", "Docs": "EventOutput represents new output from a build.\nText only contains the newly added output, not the full output so far.", "Fields": [{ "Name": "BuildID", "Docs": "", "Typewords": ["int32"] }, { "Name": "Step", "Docs": "During which the output was generated, eg `clone`, `build`.", "Typewords": ["string"] }, { "Name": "Where", "Docs": "`stdout` or `stderr`.", "Typewords": ["string"] }, { "Name": "Text", "Docs": "Lines of text written.", "Typewords": ["string"] }] },
	};
	api.parser = {
		Build: (v) => api.parse("Build", v),
		Result: (v) => api.parse("Result", v),
		Step: (v) => api.parse("Step", v),
		RepoBuilds: (v) => api.parse("RepoBuilds", v),
		Repo: (v) => api.parse("Repo", v),
		GoToolchains: (v) => api.parse("GoToolchains", v),
		Settings: (v) => api.parse("Settings", v),
		BuildStatus: (v) => api.parse("BuildStatus", v),
		VCS: (v) => api.parse("VCS", v),
		LogLevel: (v) => api.parse("LogLevel", v),
		EventRepo: (v) => api.parse("EventRepo", v),
		EventRemoveRepo: (v) => api.parse("EventRemoveRepo", v),
		EventBuild: (v) => api.parse("EventBuild", v),
		EventRemoveBuild: (v) => api.parse("EventRemoveBuild", v),
		EventOutput: (v) => api.parse("EventOutput", v),
	};
	// The Ding API lets you compile git branches, build binaries, run tests, and
	// publish binaries.
	//
	// # Server-Sent Events
	// SSE is a real-time streaming updates API using server-sent event, available at /events.
	// Query string parameter "password" is required.
	// You'll receive the following events with a HTTP GET request to `/events`, encoded as JSON:
	// - `repo`, repository was updated or created
	// - `removeRepo`, repository was removed
	// - `build`, build was updated or created
	// - `removeBuild`, build was removed
	// - `output`, new lines of output from a command for an active build
	// 
	// These types are described below, with an _event_-prefix. E.g. type _EventRepo_ describes the `repo` event.
	let defaultOptions = { slicesNullable: true, mapsNullable: true, nullableOptional: true };
	class Client {
		baseURL;
		authState;
		options;
		constructor() {
			this.authState = {};
			this.options = { ...defaultOptions };
			this.baseURL = this.options.baseURL || api.defaultBaseURL;
		}
		withAuthToken(token) {
			const c = new Client();
			c.authState.token = token;
			c.options = this.options;
			return c;
		}
		withOptions(options) {
			const c = new Client();
			c.authState = this.authState;
			c.options = { ...this.options, ...options };
			return c;
		}
		// Status checks the health of the application.
		async Status() {
			const fn = "Status";
			const paramTypes = [];
			const returnTypes = [];
			const params = [];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// BuildCreate builds a specific commit in the background, returning immediately.
		// 
		// `Commit` can be empty, in which case the origin is cloned and the checked
		// out commit is looked up.
		// 
		// Low priority builds are executed after regular builds. And only one low
		// priority build is running over all repo's.
		async BuildCreate(password, repoName, branch, commit, lowPrio) {
			const fn = "BuildCreate";
			const paramTypes = [["string"], ["string"], ["string"], ["string"], ["bool"]];
			const returnTypes = [["Build"]];
			const params = [password, repoName, branch, commit, lowPrio];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// CreateBuild exists for compatibility with older "ding kick" behaviour.
		async CreateBuild(password, repoName, branch, commit) {
			const fn = "CreateBuild";
			const paramTypes = [["string"], ["string"], ["string"], ["string"]];
			const returnTypes = [["Build"]];
			const params = [password, repoName, branch, commit];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// BuildsCreateLowPrio creates low priority builds for each repository, for the default branch.
		async BuildsCreateLowPrio(password) {
			const fn = "BuildsCreateLowPrio";
			const paramTypes = [["string"]];
			const returnTypes = [];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// BuildCancel cancels a currently running build.
		async BuildCancel(password, repoName, buildID) {
			const fn = "BuildCancel";
			const paramTypes = [["string"], ["string"], ["int32"]];
			const returnTypes = [];
			const params = [password, repoName, buildID];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// ReleaseCreate release a build.
		async ReleaseCreate(password, repoName, buildID) {
			const fn = "ReleaseCreate";
			const paramTypes = [["string"], ["string"], ["int32"]];
			const returnTypes = [["Build"]];
			const params = [password, repoName, buildID];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// RepoBuilds returns all repositories and recent build info for "active" branches.
		// A branch is active if its name is "master" or "main" (for git), "default" (for hg), or
		// "develop", or if the last build was less than 4 weeks ago. The most recent
		// build is returned.
		async RepoBuilds(password) {
			const fn = "RepoBuilds";
			const paramTypes = [["string"]];
			const returnTypes = [["[]", "RepoBuilds"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// Repo returns the named repository.
		async Repo(password, repoName) {
			const fn = "Repo";
			const paramTypes = [["string"], ["string"]];
			const returnTypes = [["Repo"]];
			const params = [password, repoName];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// Builds returns builds for a repo.
		// 
		// The Steps field of builds is cleared for transfer size.
		async Builds(password, repoName) {
			const fn = "Builds";
			const paramTypes = [["string"], ["string"]];
			const returnTypes = [["[]", "Build"]];
			const params = [password, repoName];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// RepoCreate creates a new repository.
		// If repo.UID is not null, a unique uid is assigned.
		async RepoCreate(password, repo) {
			const fn = "RepoCreate";
			const paramTypes = [["string"], ["Repo"]];
			const returnTypes = [["Repo"]];
			const params = [password, repo];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// RepoSave changes a repository.
		async RepoSave(password, repo) {
			const fn = "RepoSave";
			const paramTypes = [["string"], ["Repo"]];
			const returnTypes = [["Repo"]];
			const params = [password, repo];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// RepoClearHomedir removes the home directory this repository shares across
		// builds.
		async RepoClearHomedir(password, repoName) {
			const fn = "RepoClearHomedir";
			const paramTypes = [["string"], ["string"]];
			const returnTypes = [];
			const params = [password, repoName];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// ClearRepoHomedirs removes the home directory of all repositories.
		async ClearRepoHomedirs(password) {
			const fn = "ClearRepoHomedirs";
			const paramTypes = [["string"]];
			const returnTypes = [];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// RepoRemove removes a repository and all its builds.
		async RepoRemove(password, repoName) {
			const fn = "RepoRemove";
			const paramTypes = [["string"], ["string"]];
			const returnTypes = [];
			const params = [password, repoName];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// Build returns the build, including steps output.
		async Build(password, repoName, buildID) {
			const fn = "Build";
			const paramTypes = [["string"], ["string"], ["int32"]];
			const returnTypes = [["Build"]];
			const params = [password, repoName, buildID];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// BuildRemove removes a build completely. Both from database and all local files.
		async BuildRemove(password, buildID) {
			const fn = "BuildRemove";
			const paramTypes = [["string"], ["int32"]];
			const returnTypes = [];
			const params = [password, buildID];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// BuildCleanupBuilddir cleans up (removes) a build directory.
		// This does not remove the build itself from the database.
		async BuildCleanupBuilddir(password, repoName, buildID) {
			const fn = "BuildCleanupBuilddir";
			const paramTypes = [["string"], ["string"], ["int32"]];
			const returnTypes = [["Build"]];
			const params = [password, repoName, buildID];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// GoToolchainsListInstalled returns the installed Go toolchains (eg "go1.13.8",
		// "go1.14") in GoToolchainDir, and current "active" versions with a shortname, eg
		// "go" as "go1.14", "goprev" as "go1.13.8" and "gonext" as "go1.23rc1".
		async GoToolchainsListInstalled(password) {
			const fn = "GoToolchainsListInstalled";
			const paramTypes = [["string"]];
			const returnTypes = [["[]", "string"], ["GoToolchains"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// GoToolchainsListReleased returns all known released Go toolchains available at
		// golang.org/dl/, eg "go1.13.8", "go1.14".
		async GoToolchainsListReleased(password) {
			const fn = "GoToolchainsListReleased";
			const paramTypes = [["string"]];
			const returnTypes = [["[]", "string"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// GoToolchainInstall downloads, verifies and extracts the release Go toolchain
		// represented by goversion (eg "go1.13.8", "go1.14") into the GoToolchainDir, and
		// optionally "activates" the version under shortname ("go", "goprev", "gonext", ""; empty
		// string does nothing).
		async GoToolchainInstall(password, goversion, shortname) {
			const fn = "GoToolchainInstall";
			const paramTypes = [["string"], ["string"], ["string"]];
			const returnTypes = [];
			const params = [password, goversion, shortname];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// GoToolchainRemove removes a toolchain from the go toolchain dir.
		// It also removes shortname symlinks to this toolchain if they exists.
		async GoToolchainRemove(password, goversion) {
			const fn = "GoToolchainRemove";
			const paramTypes = [["string"], ["string"]];
			const returnTypes = [];
			const params = [password, goversion];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// GoToolchainActivate activates goversion (eg "go1.13.8", "go1.14") under the name
		// shortname ("go", "goprev" or "gonext"), by creating a symlink in the GoToolchainDir.
		async GoToolchainActivate(password, goversion, shortname) {
			const fn = "GoToolchainActivate";
			const paramTypes = [["string"], ["string"], ["string"]];
			const returnTypes = [];
			const params = [password, goversion, shortname];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// GoToolchainAutomatic looks up the latest released Go toolchains, and installs
		// the current and previous releases, and the next (release candidate) if present.
		// Then it starts low-prio builds for all repositories that have opted in to
		// automatic building on new Go toolchains.
		async GoToolchainAutomatic(password) {
			const fn = "GoToolchainAutomatic";
			const paramTypes = [["string"]];
			const returnTypes = [["bool"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// LogLevel returns the current log level.
		async LogLevel(password) {
			const fn = "LogLevel";
			const paramTypes = [["string"]];
			const returnTypes = [["LogLevel"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// LogLevelSet sets a new log level.
		async LogLevelSet(password, level) {
			const fn = "LogLevelSet";
			const paramTypes = [["string"], ["LogLevel"]];
			const returnTypes = [];
			const params = [password, level];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// Settings returns the runtime settings.
		async Settings(password) {
			const fn = "Settings";
			const paramTypes = [["string"]];
			const returnTypes = [["bool"], ["bool"], ["bool"], ["Settings"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// SettingsSave saves the runtime settings.
		async SettingsSave(password, settings) {
			const fn = "SettingsSave";
			const paramTypes = [["string"], ["Settings"]];
			const returnTypes = [];
			const params = [password, settings];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// Version returns the ding version this instance is running.
		async Version(password) {
			const fn = "Version";
			const paramTypes = [["string"]];
			const returnTypes = [["string"], ["string"], ["string"], ["string"], ["bool"]];
			const params = [password];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
		// ExampleSSE is a no-op.
		// This function only serves to include documentation for the server-sent event types.
		async ExampleSSE() {
			const fn = "ExampleSSE";
			const paramTypes = [];
			const returnTypes = [["EventRepo"], ["EventRemoveRepo"], ["EventBuild"], ["EventRemoveBuild"], ["EventOutput"]];
			const params = [];
			return await _sherpaCall(this.baseURL, this.authState, { ...this.options }, paramTypes, returnTypes, fn, params);
		}
	}
	api.Client = Client;
	api.defaultBaseURL = (function () {
		let p = location.pathname;
		if (p && p[p.length - 1] !== '/') {
			let l = location.pathname.split('/');
			l = l.slice(0, l.length - 1);
			p = '/' + l.join('/') + '/';
		}
		return location.protocol + '//' + location.host + p + 'ding/';
	})();
	// NOTE: code below is shared between github.com/mjl-/sherpaweb and github.com/mjl-/sherpats.
	// KEEP IN SYNC.
	api.supportedSherpaVersion = 1;
	// verifyArg typechecks "v" against "typewords", returning a new (possibly modified) value for JSON-encoding.
	// toJS indicate if the data is coming into JS. If so, timestamps are turned into JS Dates. Otherwise, JS Dates are turned into strings.
	// allowUnknownKeys configures whether unknown keys in structs are allowed.
	// types are the named types of the API.
	api.verifyArg = (path, v, typewords, toJS, allowUnknownKeys, types, opts) => {
		return new verifier(types, toJS, allowUnknownKeys, opts).verify(path, v, typewords);
	};
	api.parse = (name, v) => api.verifyArg(name, v, [name], true, false, api.types, defaultOptions);
	class verifier {
		types;
		toJS;
		allowUnknownKeys;
		opts;
		constructor(types, toJS, allowUnknownKeys, opts) {
			this.types = types;
			this.toJS = toJS;
			this.allowUnknownKeys = allowUnknownKeys;
			this.opts = opts;
		}
		verify(path, v, typewords) {
			typewords = typewords.slice(0);
			const ww = typewords.shift();
			const error = (msg) => {
				if (path != '') {
					msg = path + ': ' + msg;
				}
				throw new Error(msg);
			};
			if (typeof ww !== 'string') {
				error('bad typewords');
				return; // should not be necessary, typescript doesn't see error always throws an exception?
			}
			const w = ww;
			const ensure = (ok, expect) => {
				if (!ok) {
					error('got ' + JSON.stringify(v) + ', expected ' + expect);
				}
				return v;
			};
			switch (w) {
				case 'nullable':
					if (v === null || v === undefined && this.opts.nullableOptional) {
						return v;
					}
					return this.verify(path, v, typewords);
				case '[]':
					if (v === null && this.opts.slicesNullable || v === undefined && this.opts.slicesNullable && this.opts.nullableOptional) {
						return v;
					}
					ensure(Array.isArray(v), "array");
					return v.map((e, i) => this.verify(path + '[' + i + ']', e, typewords));
				case '{}':
					if (v === null && this.opts.mapsNullable || v === undefined && this.opts.mapsNullable && this.opts.nullableOptional) {
						return v;
					}
					ensure(v !== null || typeof v === 'object', "object");
					const r = {};
					for (const k in v) {
						r[k] = this.verify(path + '.' + k, v[k], typewords);
					}
					return r;
			}
			ensure(typewords.length == 0, "empty typewords");
			const t = typeof v;
			switch (w) {
				case 'any':
					return v;
				case 'bool':
					ensure(t === 'boolean', 'bool');
					return v;
				case 'int8':
				case 'uint8':
				case 'int16':
				case 'uint16':
				case 'int32':
				case 'uint32':
				case 'int64':
				case 'uint64':
					ensure(t === 'number' && Number.isInteger(v), 'integer');
					return v;
				case 'float32':
				case 'float64':
					ensure(t === 'number', 'float');
					return v;
				case 'int64s':
				case 'uint64s':
					ensure(t === 'number' && Number.isInteger(v) || t === 'string', 'integer fitting in float without precision loss, or string');
					return '' + v;
				case 'string':
					ensure(t === 'string', 'string');
					return v;
				case 'timestamp':
					if (this.toJS) {
						ensure(t === 'string', 'string, with timestamp');
						const d = new Date(v);
						if (d instanceof Date && !isNaN(d.getTime())) {
							return d;
						}
						error('invalid date ' + v);
					}
					else {
						ensure(t === 'object' && v !== null, 'non-null object');
						ensure(v.__proto__ === Date.prototype, 'Date');
						return v.toISOString();
					}
			}
			// We're left with named types.
			const nt = this.types[w];
			if (!nt) {
				error('unknown type ' + w);
			}
			if (v === null) {
				error('bad value ' + v + ' for named type ' + w);
			}
			if (api.structTypes[nt.Name]) {
				const t = nt;
				if (typeof v !== 'object') {
					error('bad value ' + v + ' for struct ' + w);
				}
				const r = {};
				for (const f of t.Fields) {
					r[f.Name] = this.verify(path + '.' + f.Name, v[f.Name], f.Typewords);
				}
				// If going to JSON also verify no unknown fields are present.
				if (!this.allowUnknownKeys) {
					const known = {};
					for (const f of t.Fields) {
						known[f.Name] = true;
					}
					Object.keys(v).forEach((k) => {
						if (!known[k]) {
							error('unknown key ' + k + ' for struct ' + w);
						}
					});
				}
				return r;
			}
			else if (api.stringsTypes[nt.Name]) {
				const t = nt;
				if (typeof v !== 'string') {
					error('mistyped value ' + v + ' for named strings ' + t.Name);
				}
				if (!t.Values || t.Values.length === 0) {
					return v;
				}
				for (const sv of t.Values) {
					if (sv.Value === v) {
						return v;
					}
				}
				error('unknown value ' + v + ' for named strings ' + t.Name);
			}
			else if (api.intsTypes[nt.Name]) {
				const t = nt;
				if (typeof v !== 'number' || !Number.isInteger(v)) {
					error('mistyped value ' + v + ' for named ints ' + t.Name);
				}
				if (!t.Values || t.Values.length === 0) {
					return v;
				}
				for (const sv of t.Values) {
					if (sv.Value === v) {
						return v;
					}
				}
				error('unknown value ' + v + ' for named ints ' + t.Name);
			}
			else {
				throw new Error('unexpected named type ' + nt);
			}
		}
	}
	const _sherpaCall = async (baseURL, authState, options, paramTypes, returnTypes, name, params) => {
		if (!options.skipParamCheck) {
			if (params.length !== paramTypes.length) {
				return Promise.reject({ message: 'wrong number of parameters in sherpa call, saw ' + params.length + ' != expected ' + paramTypes.length });
			}
			params = params.map((v, index) => api.verifyArg('params[' + index + ']', v, paramTypes[index], false, false, api.types, options));
		}
		const simulate = async (json) => {
			const config = JSON.parse(json || 'null') || {};
			const waitMinMsec = config.waitMinMsec || 0;
			const waitMaxMsec = config.waitMaxMsec || 0;
			const wait = Math.random() * (waitMaxMsec - waitMinMsec);
			const failRate = config.failRate || 0;
			return new Promise((resolve, reject) => {
				if (options.aborter) {
					options.aborter.abort = () => {
						reject({ message: 'call to ' + name + ' aborted by user', code: 'sherpa:aborted' });
						reject = resolve = () => { };
					};
				}
				setTimeout(() => {
					const r = Math.random();
					if (r < failRate) {
						reject({ message: 'injected failure on ' + name, code: 'server:injected' });
					}
					else {
						resolve();
					}
					reject = resolve = () => { };
				}, waitMinMsec + wait);
			});
		};
		// Only simulate when there is a debug string. Otherwise it would always interfere
		// with setting options.aborter.
		let json = '';
		try {
			json = window.localStorage.getItem('sherpats-debug') || '';
		}
		catch (err) { }
		if (json) {
			await simulate(json);
		}
		const fn = (resolve, reject) => {
			let resolve1 = (v) => {
				resolve(v);
				resolve1 = () => { };
				reject1 = () => { };
			};
			let reject1 = (v) => {
				if ((v.code === 'user:noAuth' || v.code === 'user:badAuth') && options.login) {
					const login = options.login;
					if (!authState.loginPromise) {
						authState.loginPromise = new Promise((aresolve, areject) => {
							login(v.code === 'user:badAuth' ? (v.message || '') : '')
								.then((token) => {
								authState.token = token;
								authState.loginPromise = undefined;
								aresolve();
							}, (err) => {
								authState.loginPromise = undefined;
								areject(err);
							});
						});
					}
					authState.loginPromise
						.then(() => {
						fn(resolve, reject);
					}, (err) => {
						reject(err);
					});
					return;
				}
				reject(v);
				resolve1 = () => { };
				reject1 = () => { };
			};
			const url = baseURL + name;
			const req = new window.XMLHttpRequest();
			if (options.aborter) {
				options.aborter.abort = () => {
					req.abort();
					reject1({ code: 'sherpa:aborted', message: 'request aborted' });
				};
			}
			req.open('POST', url, true);
			if (options.csrfHeader && authState.token) {
				req.setRequestHeader(options.csrfHeader, authState.token);
			}
			if (options.timeoutMsec) {
				req.timeout = options.timeoutMsec;
			}
			req.onload = () => {
				if (req.status !== 200) {
					if (req.status === 404) {
						reject1({ code: 'sherpa:badFunction', message: 'function does not exist' });
					}
					else {
						reject1({ code: 'sherpa:http', message: 'error calling function, HTTP status: ' + req.status });
					}
					return;
				}
				let resp;
				try {
					resp = JSON.parse(req.responseText);
				}
				catch (err) {
					reject1({ code: 'sherpa:badResponse', message: 'bad JSON from server' });
					return;
				}
				if (resp && resp.error) {
					const err = resp.error;
					reject1({ code: err.code, message: err.message });
					return;
				}
				else if (!resp || !resp.hasOwnProperty('result')) {
					reject1({ code: 'sherpa:badResponse', message: "invalid sherpa response object, missing 'result'" });
					return;
				}
				if (options.skipReturnCheck) {
					resolve1(resp.result);
					return;
				}
				let result = resp.result;
				try {
					if (returnTypes.length === 0) {
						if (result) {
							throw new Error('function ' + name + ' returned a value while prototype says it returns "void"');
						}
					}
					else if (returnTypes.length === 1) {
						result = api.verifyArg('result', result, returnTypes[0], true, true, api.types, options);
					}
					else {
						if (result.length != returnTypes.length) {
							throw new Error('wrong number of values returned by ' + name + ', saw ' + result.length + ' != expected ' + returnTypes.length);
						}
						result = result.map((v, index) => api.verifyArg('result[' + index + ']', v, returnTypes[index], true, true, api.types, options));
					}
				}
				catch (err) {
					let errmsg = 'bad types';
					if (err instanceof Error) {
						errmsg = err.message;
					}
					reject1({ code: 'sherpa:badTypes', message: errmsg });
				}
				resolve1(result);
			};
			req.onerror = () => {
				reject1({ code: 'sherpa:connection', message: 'connection failed' });
			};
			req.ontimeout = () => {
				reject1({ code: 'sherpa:timeout', message: 'request timeout' });
			};
			req.setRequestHeader('Content-Type', 'application/json');
			try {
				req.send(JSON.stringify({ params: params }));
			}
			catch (err) {
				reject1({ code: 'sherpa:badData', message: 'cannot marshal to JSON' });
			}
		};
		return await new Promise(fn);
	};
})(api || (api = {}));
let rootElem;
let crumbElem = dom.span();
let updateElem = dom.span();
let pageElem = dom.div(style({ padding: '1em' }), dom.div(style({ textAlign: 'center' }), 'Loading...'));
const client = new api.Client();
const colors = {
	green: '#66ac4c',
	blue: 'rgb(70, 158, 211)',
	red: 'rgb(228, 77, 52)',
	gray: 'rgb(138, 138, 138)',
};
let favicon = dom.link(attr.rel('icon'), attr.href('favicon.ico')); // attr.href changed for some build states
let favicons = {
	default: 'favicon.ico',
	green: 'favicon-green.png',
	red: 'favicon-red.png',
	gray: 'favicon-gray.png',
};
const setFavicon = (href) => {
	favicon.setAttribute('href', href);
};
const buildSetFavicon = (b) => {
	if (!b.Finish) {
		setFavicon(favicons.gray);
	}
	else if (b.Status !== api.BuildStatus.StatusSuccess) {
		setFavicon(favicons.red);
	}
	else {
		setFavicon(favicons.green);
	}
};
const link = (href, anchor) => dom.a(attr.href(href), anchor);
class Stream {
	subscribers = [];
	send(e) {
		this.subscribers.forEach(fn => fn(e));
	}
	subscribe(fn) {
		this.subscribers.push(fn);
		return () => {
			this.subscribers = this.subscribers.filter(s => s !== fn);
		};
	}
}
const streams = {
	repo: new Stream(),
	removeRepo: new Stream(),
	build: new Stream(),
	removeBuild: new Stream(),
	output: new Stream(),
};
let sseElem = dom.span('Disconnected from live updates.'); // Shown in UI next to logout button.
let eventSource; // We initialize it after first success API call.
let allowReconnect = false;
const initEventSource = () => {
	// todo: update ui that we are busy connecting
	dom._kids(sseElem, 'Connecting...');
	eventSource = new window.EventSource('events?password=' + encodeURIComponent(password));
	eventSource.addEventListener('open', function () {
		allowReconnect = true;
		dom._kids(sseElem);
	});
	eventSource.addEventListener('error', function (event) {
		console.log('sse connection error', event);
		if (allowReconnect) {
			allowReconnect = false;
			initEventSource();
		}
		else {
			// todo: on window focus, we could do another reconnect attempt, timethrottled.
			dom._kids(sseElem, 'Connection error for live updates. ', dom.clickbutton('Reconnect', function click() {
				dom._kids(sseElem);
				initEventSource();
			}));
		}
	});
	eventSource.addEventListener('repo', (e) => streams.repo.send(api.parser.EventRepo(JSON.parse(e.data))));
	eventSource.addEventListener('removeRepo', (e) => streams.removeRepo.send(api.parser.EventRemoveRepo(JSON.parse(e.data))));
	eventSource.addEventListener('build', (e) => streams.build.send(api.parser.EventBuild(JSON.parse(e.data))));
	eventSource.addEventListener('removeBuild', (e) => streams.removeBuild.send(api.parser.EventRemoveBuild(JSON.parse(e.data))));
	eventSource.addEventListener('output', (e) => streams.output.send(api.parser.EventOutput(JSON.parse(e.data))));
};
// Atexit helps run cleanup code when a page is unloaded. A page has an atexit to
// which functions can be added. Pages that can rerender parts of their contents
// can create a new atexit for a part, register the cleanup function with their
// page (or higher level atexit), and call run to cleanup before rerendering.
class Atexit {
	fns = [];
	run() {
		for (const fn of this.fns) {
			fn();
		}
		this.fns = [];
	}
	add(fn) {
		this.fns.push(fn);
	}
	age(start, end) {
		const [elem, close] = age0(false, start, end);
		this.add(close);
		return elem;
	}
	ageMins(start, end) {
		const [elem, close] = age0(true, start, end);
		this.add(close);
		return elem;
	}
}
// Page is a loaded page, used to clean up references to event streams and timers.
class Page {
	atexit = new Atexit();
	updateRoot; // Box holding status about SSE connection.
	newAtexit() {
		const atexit = new Atexit();
		this.atexit.add(() => atexit.run());
		return atexit;
	}
	cleanup() {
		this.atexit.run();
	}
	subscribe(s, fn) {
		this.atexit.add(s.subscribe(fn));
	}
}
let loginPromise;
let password = '';
// authed calls fn and awaits the promise it returns. If the promise fails with an
// error object with .code 'user:badAuth', it shows a popup for a password, then
// calls the function again through authed for any password retries.
const authed = async (fn, elem) => {
	const overlay = dom.div(style({ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 2, backgroundColor: '#ffffff00' }));
	document.body.append(overlay);
	pageElem.classList.toggle('loading', true);
	if (elem) {
		elem.disabled = true;
	}
	const done = () => {
		overlay.remove();
		pageElem.classList.toggle('loading', false);
		if (elem) {
			elem.disabled = false;
		}
	};
	try {
		const r = await fn();
		done();
		if (!eventSource) {
			initEventSource();
		}
		return r;
	}
	catch (err) {
		done();
		if (err.code !== 'user:noAuth') {
			alert('Error: ' + err.message);
		}
		if (err.code === 'user:badAuth' || err.code === 'user:noAuth') {
			if (!loginPromise) {
				let passwordElem;
				loginPromise = new Promise((resolve) => {
					const close = popupOpts(true, dom.h1('Login'), dom.form(function submit(e) {
						e.stopPropagation();
						e.preventDefault();
						password = passwordElem.value;
						try {
							window.localStorage.setItem('dingpassword', password);
						}
						catch (err) {
							console.log('setting session storage', err);
						}
						resolve();
						close();
					}, dom.fieldset(dom.div(dom.label(dom.div('Password'), passwordElem = dom.input(attr.type('password'), attr.required('')))), dom.br(), dom.div(dom.submitbutton('Login')))));
					passwordElem.focus();
				});
				await loginPromise;
				loginPromise = undefined;
			}
			else {
				await loginPromise;
			}
			return await authed(fn, elem);
		}
		throw err;
	}
};
const formatCoverage = (repo, b) => {
	const anchor = b.Coverage ? (Math.round(b.Coverage) + '%') : 'report';
	if (b.CoverageReportFile && !b.BuilddirRemoved) {
		return dom.a(attr.href('dl/file/' + encodeURIComponent(repo.Name) + '/' + b.ID + '/' + b.CoverageReportFile), anchor);
	}
	return anchor === 'report' ? '' : anchor;
};
const age0 = (mins, start, end) => {
	const second = 1;
	const minute = 60 * second;
	const hour = 60 * minute;
	const day = 24 * hour;
	const week = 7 * day;
	const year = 365 * day;
	const periods = [year, week, day, hour, minute, second];
	const suffix = ['y', 'w', 'd', 'h', 'm', 's'];
	const elem = dom.span(attr.title(start.toString()));
	let id = 0;
	const cleanup = () => {
		if (id) {
			window.clearTimeout(id);
			id = 0;
		}
	};
	const set = () => {
		const e = (end || new Date()).getTime() / 1000;
		let t = e - start.getTime() / 1000;
		let nextSecs = 0;
		let s = '';
		for (let i = 0; i < periods.length; i++) {
			const p = periods[i];
			if (t >= 2 * p || i === periods.length - 1 || mins && p === minute) {
				if (p == second && t < 10 * second) {
					nextSecs = 0.1;
					s = t.toFixed(1) + 's';
					break;
				}
				const n = Math.round(t / p);
				s = '' + n + suffix[i];
				const prev = Math.floor(t / p);
				nextSecs = Math.ceil((prev + 1) * p - t);
				break;
			}
		}
		if (!mins && !end) {
			s += '...';
		}
		dom._kids(elem, s);
		// note: Cannot have delays longer than 24.8 days due to storage as 32 bit in
		// browsers. Session is likely closed/reloaded/refreshed before that time anyway.
		return Math.min(nextSecs, 14 * 24 * 3600);
	};
	if (end) {
		set();
		return [elem, cleanup];
	}
	const refresh = () => {
		const nextSecs = set();
		id = window.setTimeout(refresh, nextSecs * 1000);
	};
	refresh();
	return [elem, cleanup];
};
const formatSize = (size) => (size / (1024 * 1024)).toFixed(1) + 'm';
const formatBuildSize = (b) => dom.span(attr.title('Disk usage of build directory (including checkout directory), and optional difference in size of (reused) home directory'), formatSize(b.DiskUsage) + (b.HomeDiskUsageDelta ? (b.HomeDiskUsageDelta > 0 ? '+' : '') + formatSize(b.HomeDiskUsageDelta) : ''));
const statusColor = (b) => {
	if (b.ErrorMessage || b.Finish && b.Status !== api.BuildStatus.StatusSuccess) {
		return colors.red;
	}
	else if (b.Released) {
		return colors.blue;
	}
	else if (b.Finish) {
		return colors.green;
	}
	else {
		return colors.gray;
	}
};
const buildStatus = (b) => {
	let s = b.Status;
	if (b.Status === api.BuildStatus.StatusNew && b.LowPrio) {
		s += '↓';
	}
	return dom.span(s, style({ fontSize: '.9em', color: 'white', backgroundColor: statusColor(b), padding: '0 .2em', borderRadius: '.15em' }));
};
const buildErrmsg = (b) => {
	let msg = b.ErrorMessage;
	if (b.ErrorMessage && b.LastLine) {
		msg += ', "' + b.LastLine + '"';
	}
	return msg ? dom.span(style({ maxWidth: '40em', display: 'inline-block' }), msg) : [];
};
const popupOpts = (opaque, ...kids) => {
	const origFocus = document.activeElement;
	const close = () => {
		if (!root.parentNode) {
			return;
		}
		root.remove();
		if (origFocus && origFocus instanceof HTMLElement && origFocus.parentNode) {
			origFocus.focus();
		}
	};
	let content;
	const root = dom.div(style({ position: 'fixed', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: opaque ? '#ffffff' : 'rgba(0, 0, 0, 0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: opaque ? 3 : 1 }), opaque ? [] : [
		function keydown(e) {
			if (e.key === 'Escape') {
				e.stopPropagation();
				close();
			}
		},
		function click(e) {
			e.stopPropagation();
			close();
		},
	], content = dom.div(attr.tabindex('0'), style({ backgroundColor: 'white', borderRadius: '.25em', padding: '1em', boxShadow: '0 0 20px rgba(0, 0, 0, 0.1)', border: '1px solid #ddd', maxWidth: '95vw', overflowX: 'auto', maxHeight: '95vh', overflowY: 'auto' }), function click(e) {
		e.stopPropagation();
	}, kids));
	document.body.appendChild(root);
	content.focus();
	return close;
};
const popup = (...kids) => popupOpts(false, ...kids);
const popupRepoAdd = async (haveBubblewrap, haveGoToolchainDir) => {
	let vcs;
	let origin;
	let originBox;
	let originInput;
	let originTextarea;
	let name;
	let defaultBranch;
	let reuseUID;
	let bubblewrap;
	let bubblewrapNoNet;
	let buildOnUpdatedToolchain;
	let goauto;
	let gocur;
	let goprev;
	let gonext;
	let fieldset;
	let branchChanged = false;
	let nameChanged = false;
	const originTextareaBox = dom.div(originTextarea = dom.textarea(attr.required(''), attr.rows('5'), style({ width: '100%' })), dom.div('Script that clones a repository into checkout/$DING_CHECKOUTPATH.'), dom.div('Typically starts with "#!/bin/sh".'), dom.div('It must print a line of the form "commit: ...".'), dom.br());
	const vcsChanged = function change() {
		if (!branchChanged) {
			if (vcs.value === 'git') {
				defaultBranch.value = 'main';
			}
			else if (vcs.value === 'mercurial') {
				defaultBranch.value = 'default';
			}
			else if (vcs.value === 'command') {
				defaultBranch.value = '';
			}
		}
		if (vcs.value !== 'command') {
			const n = dom.div(originInput);
			originBox.replaceWith(n);
			originBox = n;
			origin = originInput;
		}
		else {
			originBox.replaceWith(originTextareaBox);
			originBox = originTextareaBox;
			origin = originTextarea;
		}
	};
	const close = popup(dom.h1('New repository'), dom.form(async function submit(e) {
		e.stopPropagation();
		e.preventDefault();
		const repo = {
			Name: name.value,
			VCS: vcs.value,
			Origin: origin.value,
			DefaultBranch: defaultBranch.value,
			UID: reuseUID.checked ? 1 : null,
			CheckoutPath: name.value,
			Bubblewrap: bubblewrap.checked,
			BubblewrapNoNet: bubblewrapNoNet.checked,
			BuildOnUpdatedToolchain: buildOnUpdatedToolchain.checked,
			WebhookSecret: '',
			AllowGlobalWebhookSecrets: false,
			BuildScript: '',
			HomeDiskUsage: 0,
			GoAuto: goauto.checked,
			GoCur: gocur.checked,
			GoPrev: goprev.checked,
			GoNext: gonext.checked,
		};
		const r = await authed(() => client.RepoCreate(password, repo), fieldset);
		location.hash = '#repo/' + encodeURIComponent(r.Name);
		close();
	}, fieldset = dom.fieldset(dom.div(style({ display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top' }), dom.span('VCS', attr.title('Clones are run as the configured ding user, not under a unique/reused UID. After cloning, file permissions are fixed up. Configure an .ssh/config and/or ssh keys in the home directory of the ding user.')), vcs = dom.select(dom.option('git'), dom.option('mercurial'), dom.option('command'), vcsChanged), 'Origin', originBox = dom.div(originInput = origin = dom.input(attr.required(''), attr.placeholder('https://... or ssh://... or user@host:path.git'), style({ width: '100%' }), function keyup() {
		if (nameChanged) {
			return;
		}
		let t = origin.value.split('/');
		let s = t[t.length - 1] || t[t.length - 2] || '';
		s = s.replace(/\.git$/, '');
		name.value = s;
	})), 'Name', name = dom.input(attr.required(''), function change() { nameChanged = true; }), dom.div('Default branch', style({ whiteSpace: 'nowrap' })), defaultBranch = dom.input(attr.value('main'), attr.placeholder('main, master, default'), function change() { branchChanged = true; }), dom.div(), dom.label(reuseUID = dom.input(attr.type('checkbox'), attr.checked('')), ' Reuse $HOME and UID for builds for this repo', attr.title('By reusing $HOME and running builds for this repository under the same UID, build caches can be used. This typically leads to faster builds but reduces isolation of builds.')), dom.div(), dom.label(bubblewrap = dom.input(attr.type('checkbox'), haveBubblewrap ? attr.checked('') : []), ' Run build script in bubblewrap, with limited system access', attr.title('Only available on Linux, with bubblewrap (bwrap) installed. Commands are run in a new mount namespace with access to system directories like /bin /lib /usr, and to the ding build, home and toolchain directories.')), dom.div(), dom.label(bubblewrapNoNet = dom.input(attr.type('checkbox'), haveBubblewrap ? attr.checked('') : []), ' Prevent network access from build script. Only active if bubblewrap is active.', attr.title('Hide network interfaces from the build script. Only a loopback device is available.')), dom.div('Build for Go toolchains', style({ whiteSpace: 'nowrap' }), attr.title('The build script will be run for each of the selected Go toolchains. The short name (go, goprev, gonext) is set in $DING_GOTOOLCHAIN. If this build was triggered due to a new Go toolchain being installed, the variable $DING_NEWGOTOOLCHAIN is set.')), dom.div(dom.label(goauto = dom.input(attr.type('checkbox'), haveGoToolchainDir ? attr.checked('') : '', function change() {
		if (goauto.checked) {
			gocur.checked = false;
			goprev.checked = false;
			gonext.checked = false;
		}
	}), ' Automatic', attr.title('Build for each of the available Go toolchains, go/goprev/gonext. At least one must be found or the build will fail.')), ' ', dom.label(gocur = dom.input(attr.type('checkbox'), function change() { goauto.checked = false; }), ' Go latest', attr.title('Latest patch version of latest stable Go toolchain version.')), ' ', dom.label(goprev = dom.input(attr.type('checkbox'), function change() { goauto.checked = false; }), ' Go previous', attr.title('Latest patch version of Go toolchain minor version before the latest stable.')), ' ', dom.label(gonext = dom.input(attr.type('checkbox'), function change() { goauto.checked = false; }), ' Go next', attr.title('Release candidate of Go toolchain, if available.')), ' '), dom.div(), dom.label(buildOnUpdatedToolchain = dom.input(attr.type('checkbox'), attr.checked('')), ' Schedule a low-priority build when new toolchains are automatically installed.')), dom.br(), dom.p('The build script can be configured after creating.'), dom.div(style({ textAlign: 'right' }), dom.submitbutton('Add')))));
	originInput.focus();
};
const pageHome = async () => {
	const page = new Page();
	let [rbl0, [, , , , haveBubblewrap]] = await authed(() => Promise.all([
		client.RepoBuilds(password),
		client.Version(password),
	]));
	let rbl = rbl0 || [];
	const rblFavicon = () => {
		let busy = false;
		for (const rb of rbl) {
			for (const b of (rb.Builds || [])) {
				if (!b.Finish) {
					busy = true;
					break;
				}
			}
		}
		setFavicon(busy ? favicons.gray : favicons.default);
	};
	dom._kids(crumbElem, 'Home');
	document.title = 'Ding - Repos';
	rblFavicon();
	const atexit = page.newAtexit();
	const render = () => {
		atexit.run();
		dom._kids(pageElem, dom.div(style({ marginBottom: '1ex' }), link('#gotoolchains', 'Go Toolchains'), ' ', link('#settings', 'Settings'), ' '), dom.div(dom.clickbutton('Add repo', attr.title('Add new repository, to build.'), async function click() {
			const [, , haveGoToolchainDir] = await authed(() => client.Settings(password));
			popupRepoAdd(haveBubblewrap, haveGoToolchainDir);
		}), ' ', dom.clickbutton('Clear homedirs', attr.title('Remove home directories for all repositories that reuse home directories across builds. Cache in such directories can grow over time, consuming quite some disk space.'), async function click(e) {
			if (!confirm('Are you sure?')) {
				return;
			}
			await authed(() => client.ClearRepoHomedirs(password), e.target);
		}), ' ', dom.clickbutton('Build all lowprio', attr.title('Schedule builds for all repositories, but at low priority.'), async function click(e) {
			await authed(() => client.BuildsCreateLowPrio(password), e.target);
		})), dom.table(dom._class('striped', 'wide'), dom.thead(dom.tr(['Repo', 'Build ID', 'Status', 'Duration', 'Branch', 'Version', 'Coverage', 'Disk usage', 'Home disk usage', 'Age'].map(s => dom.th(s)), dom.th(style({ textAlign: 'left' }), 'Error'))), dom.tbody(rbl.length === 0 ? dom.tr(dom.td(attr.colspan('10'), 'No repositories', style({ textAlign: 'left' }))) : [], rbl.map(rb => {
			if ((rb.Builds || []).length === 0) {
				return dom.tr(dom.td(link('#repo/' + encodeURIComponent(rb.Repo.Name), rb.Repo.Name)));
			}
			return (rb.Builds || []).map((b, i) => dom.tr(i === 0 ? dom.td(link('#repo/' + encodeURIComponent(rb.Repo.Name), rb.Repo.Name), attr.rowspan('' + (rb.Builds || []).length)) : [], dom.td(link('#repo/' + encodeURIComponent(rb.Repo.Name) + '/build/' + b.ID, '' + b.ID)), dom.td(buildStatus(b)), dom.td(b.Start ? atexit.age(b.Start, b.Finish || undefined) : ''), dom.td(b.Branch), dom.td(b.Version, b.CommitHash ? attr.title('Commit ' + b.CommitHash) : []), dom.td(formatCoverage(rb.Repo, b)), dom.td(formatBuildSize(b)), dom.td(rb.Repo.UID ? dom.span(formatSize(rb.Repo.HomeDiskUsage), attr.title('Of reused home directory')) : []), dom.td(atexit.ageMins(b.Created, undefined)), dom.td(style({ textAlign: 'left' }), buildErrmsg(b))));
		}))));
	};
	render();
	page.subscribe(streams.build, (e) => {
		const rb = rbl.find(rb => rb.Repo.Name === e.Build.RepoName);
		if (!rb) {
			return;
		}
		const builds = rb.Builds || [];
		const i = builds.findIndex(b => b.ID === e.Build.ID);
		if (i < 0) {
			builds.unshift(e.Build);
		}
		else {
			builds[i] = e.Build;
		}
		rb.Builds = builds;
		rblFavicon();
		render();
	});
	page.subscribe(streams.removeBuild, (e) => {
		const rb = rbl.find(rb => rb.Repo.Name === e.RepoName);
		if (!rb) {
			return;
		}
		rb.Builds = (rb.Builds || []).filter(b => b.ID !== e.BuildID);
		rblFavicon();
		render();
	});
	page.subscribe(streams.repo, (ev) => {
		for (const rb of rbl) {
			if (rb.Repo.Name == ev.Repo.Name) {
				rb.Repo = ev.Repo;
				render();
				return;
			}
		}
		rbl.unshift({ Repo: ev.Repo, Builds: [] });
		render();
	});
	page.subscribe(streams.removeRepo, (ev) => {
		rbl = rbl.filter(rb => rb.Repo.Name !== ev.RepoName);
		rblFavicon();
		render();
	});
	return page;
};
const pageGoToolchains = async () => {
	const page = new Page();
	const [available0, [installed0, active0]] = await authed(() => Promise.all([
		client.GoToolchainsListReleased(password),
		client.GoToolchainsListInstalled(password),
	]));
	let available = available0 || [];
	let installed = installed0 || [];
	let active = active0 || [];
	dom._kids(crumbElem, link('#', 'Home'), ' / ', 'Go Toolchains');
	document.title = 'Ding - Go Toolchains';
	setFavicon(favicons.default);
	const render = () => {
		const groups = [];
		for (const s of available) {
			const t = s.split('.');
			if (t.length === 1) {
				groups.push([s]);
				continue;
			}
			const minor = parseInt(t[1]);
			const prefix = t[0] + '.' + minor;
			if (groups.length > 0 && groups[groups.length - 1][0].startsWith(prefix)) {
				groups[groups.length - 1].push(s);
			}
			else {
				groups.push([s]);
			}
		}
		let gocur;
		let goprev;
		let gonext;
		dom._kids(pageElem, dom.p('Go toolchains can easily be installed in the toolchains directory set in the configuration file. Build scripts can add $DING_TOOLCHAINDIR/<goversion>/bin to their $PATH.'), dom.h1('Current and previous Go toolchains'), dom.p('The current/previous/next (release candidate) Go toolchains are available through $DING_TOOLCHAINDIR/{go,goprev,gonext}/bin.'), dom.table(dom.tr(dom.td('Current'), dom.td(dom.form(async function submit(e) {
			e.stopPropagation();
			e.preventDefault();
			await authed(() => client.GoToolchainActivate(password, gocur.value, 'go'));
			active.Go = gocur.value;
			render();
		}, dom.fieldset(gocur = dom.select(dom.option('(none)', attr.value('')), installed.map(s => dom.option(s, active.Go === s ? attr.selected('') : []))), ' ', dom.submitbutton('Set', attr.title('Set Go toolchain as "go"')))))), dom.tr(dom.td('Previous'), dom.td(dom.form(async function submit(e) {
			e.stopPropagation();
			e.preventDefault();
			await authed(() => client.GoToolchainActivate(password, goprev.value, 'goprev'));
			active.GoPrev = goprev.value;
			render();
		}, dom.fieldset(goprev = dom.select(dom.option('(none)', attr.value('')), installed.map(s => dom.option(s, active.GoPrev === s ? attr.selected('') : []))), ' ', dom.submitbutton('Set', attr.title('Set Go toolchain as "goprev"')))))), dom.tr(dom.td('Next'), dom.td(dom.form(async function submit(e) {
			e.stopPropagation();
			e.preventDefault();
			await authed(() => client.GoToolchainActivate(password, gonext.value, 'gonext'));
			active.GoNext = gonext.value;
			render();
		}, dom.fieldset(gonext = dom.select(dom.option('(none)', attr.value('')), installed.map(s => dom.option(s, active.GoNext === s ? attr.selected('') : []))), ' ', dom.submitbutton('Set', attr.title('Set Go toolchain as "gonext"'))))))), dom.br(), dom.div(dom.clickbutton('Automatically update toolchains', attr.title('If new toolchains are installed, low prio builds are automatically scheduled for repositories that have opted in.'), async function click(e) {
			await authed(() => client.GoToolchainAutomatic(password), e.target);
			const [available0, [installed0, active0]] = await authed(() => Promise.all([
				client.GoToolchainsListReleased(password),
				client.GoToolchainsListInstalled(password),
			]));
			available = available0 || [];
			installed = installed0 || [];
			active = active0 || [];
			render();
		})), dom.br(), dom.h1('Released and installed toolchains'), dom.div(dom.ul(style({ lineHeight: '1.75' }), groups.map(g => dom.li(g.map(s => [
			installed.includes(s) ? dom.span(s, ' ', dom.clickbutton('-', attr.title('Remove toolchain'), async function click(e) {
				await authed(() => client.GoToolchainRemove(password, s), e.target);
				installed = installed.filter(i => i !== s);
				render();
			})) : dom.clickbutton(s, attr.title('Install this toolchain'), async function click(e) {
				await authed(() => client.GoToolchainInstall(password, s, ''), e.target);
				installed.unshift(s);
				render();
			}),
			' ',
		]))))));
	};
	render();
	return page;
};
const pageSettings = async () => {
	const page = new Page();
	const [loglevel, [isolationEnabled, mailEnabled, haveGoToolchainDir, settings]] = await authed(() => Promise.all([
		client.LogLevel(password),
		client.Settings(password),
	]));
	let loglevelElem;
	let loglevelFieldset;
	let notifyEmailAddrs;
	let runPrefix;
	let environment;
	let automaticGoToolchains;
	let githubSecret;
	let giteaSecret;
	let bitbucketSecret;
	let fieldset;
	dom._kids(crumbElem, link('#', 'Home'), ' / ', 'Settings');
	document.title = 'Ding - Settings';
	setFavicon(favicons.default);
	dom._kids(pageElem, isolationEnabled ? dom.p('Each repository and potentially build is isolated to run under a unique uid.') : dom.p('NOTE: Repositories and builds are NOT isolated to run under a unique uid. You may want to enable isolated builds in the configuration file (requires restart).'), mailEnabled ? [] : dom.p('NOTE: No SMTP server is configured for outgoing emails, no email will be sent for broken/fixed builds.'), dom.div(dom.form(async function submit(e) {
		e.preventDefault();
		e.stopPropagation();
		await authed(() => client.LogLevelSet(password, loglevelElem.value), loglevelFieldset);
	}, loglevelFieldset = dom.fieldset(dom.label('Log level ', loglevelElem = dom.select(['debug', 'info', 'warn', 'error'].map(s => dom.option(s, loglevel == s ? attr.selected('') : []))), ' ', dom.submitbutton('Set'))))), dom.br(), dom.form(async function submit(e) {
		e.preventDefault();
		e.stopPropagation();
		settings.NotifyEmailAddrs = notifyEmailAddrs.value.split(',').map(s => s.trim()).filter(s => !!s);
		settings.RunPrefix = runPrefix.value.split(' ').map(s => s.trim()).filter(s => !!s);
		settings.Environment = environment.value.split('\n').map(s => s.trim()).filter(s => !!s);
		settings.AutomaticGoToolchains = automaticGoToolchains.checked;
		settings.GithubWebhookSecret = githubSecret.value;
		settings.GiteaWebhookSecret = giteaSecret.value;
		settings.BitbucketWebhookSecret = bitbucketSecret.value;
		await authed(() => client.SettingsSave(password, settings), fieldset);
	}, 
	// autocomplete=off seems to be ignored by firefox, which also isn't smart enough
	// to realize it doesn't make sense to store a password when there are 3 present in
	// a form...
	attr.autocomplete('off'), fieldset = dom.fieldset(dom.div(style({ display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top', maxWidth: '50em' }), dom.div('Notify email addresses', style({ whiteSpace: 'nowrap' }), attr.title('Comma-separated list of email address that will receive notifications when a build breaks or is fixed and a repository does not have its own addresses to notify configured.')), notifyEmailAddrs = dom.input(attr.value((settings.NotifyEmailAddrs || []).join(', ')), attr.placeholder('user@example.org, other@example.org')), dom.div('Clone and build command prefix', style({ whiteSpace: 'nowrap' }), attr.title('Can be used to run at lower priority and with timeout, e.g. "nice ionice -c 3 timeout 300s"')), runPrefix = dom.input(attr.value((settings.RunPrefix || []).join(' '))), dom.div('Additional environment variables', style({ whiteSpace: 'nowrap' }), attr.title('Of the form key=value, one per line.')), environment = dom.textarea((settings.Environment || []).map(s => s + '\n').join(''), attr.placeholder('key=value\nkey=value\n...'), attr.rows('' + Math.max(8, (settings.Environment || []).length + 1))), dom.div(), dom.label(automaticGoToolchains = dom.input(attr.type('checkbox'), settings.AutomaticGoToolchains ? attr.checked('') : []), ' Automatic Go toolchain management', attr.title('Check once per day if new Go toolchains have been released, and automatically install them and update the go/goprev/gonext symlinks, and schedule low priority builds for repositories that have opted in.' + !haveGoToolchainDir ? ' Warning: No Go toolchain directory is configured in the configuration file.' : '')), dom.div(style({ gridColumn: '1 / 3' }), 'Global webhook secrets (deprecated)', dom.p('For new repositories, unique webhooks are assigned to each repository. While global secrets are still configured, they will be accepted to start builds on all older repositories.')), dom.div('Github webhook secret', style({ whiteSpace: 'nowrap' })), githubSecret = dom.input(attr.value(settings.GithubWebhookSecret), attr.type('password'), attr.autocomplete('off')), dom.div('Gitea webhook secret', style({ whiteSpace: 'nowrap' })), giteaSecret = dom.input(attr.value(settings.GiteaWebhookSecret), attr.type('password'), attr.autocomplete('off')), dom.div('Bitbucket webhook secret', style({ whiteSpace: 'nowrap' })), bitbucketSecret = dom.input(attr.value(settings.BitbucketWebhookSecret), attr.type('password'), attr.autocomplete('off'))), dom.br(), dom.submitbutton('Save'))));
	return page;
};
const pageDocs = async () => {
	const page = new Page();
	const [version, goos, goarch, goversion] = await authed(() => client.Version(password));
	dom._kids(crumbElem, link('#', 'Home'), ' / Docs');
	document.title = 'Ding - Docs';
	setFavicon(favicons.default);
	dom._kids(pageElem, dom.h1('Introduction'), dom.p("Ding is a minimalistic build server for internal use. The goal is to make it easy to build software projects in an isolated environment, ensuring it also works on other people's machines. Ding clones a git or mercurial repository, or runs a custom shell script to clone a project, and runs a shell script to build the software. The shell script should output certain lines that ding recognizes, to find build results, test coverage, etc."), dom.h1('Notifications'), dom.p('Ding can be configured to send a notification email if a repo breaks (failed build) or is repaired again (successful build after previous failure)'), dom.h1('Webhooks'), dom.p('For each project to build, first configure a repository and a build script. Optionally configure the code repository to call a ding webhook to start a build. For git, this can be done with post-receive shell script in .git/hooks, or through various settings in web apps like gitea, github and bitbucket. For custom scripts, run ', dom.tt('ding kick baseURL repoName branch commit < password-file'), ' to start a build, where baseURL could be http://localhost:6084 (for default settings), and password is what you use for logging in. For externally-defined webhook formats, ensure the ding webhook listener is publicly accessible (e.g. through a reverse proxy), and configure these paths for the respective services: ', dom.tt('https://.../gitea/<repo>'), ', ', dom.tt('https://.../github/<repo>'), ' or ', dom.tt('https://.../bitbucket/<repo>/<secret>'), '. Gitea includes a "secret" in an Authorization header, github signs its request payload, for bitbucket you must include a secret value in the URL they send the webhook too. These secrets must be configured in the ding configuration file.'), dom.h1('Authentication'), dom.p('Ding only has simple password-based authentication, with a single password for the entire system. Everyone with the password can see all repositories, builds and scripts, and modify all data.'), dom.h1('Go toolchains'), dom.p('Ding has builtin functionality for downloading Go toolchains for use in builds.'), dom.h1('API'), dom.p('Ding has a simple HTTP/JSON-based API, see ', link('ding/', 'Ding API'), '.'), dom.h1('Files and directories'), dom.p('Ding stores all files for repositories, builds, releases and home directories in its "data" directory:'), dom.pre(`
data/
	build/<repoName>/<buildID>/		  ($DING_BUILDDIR during builds)
		checkout/$DING_CHECKOUTPATH/  (working directory for build.sh)
		scripts/
			build.sh				  (copied from database before build)
		output/
			{clone,build}.{stdout,stderr,output,nsec}
		home/						  (for builds with unique $HOME/uid)
		dl/							  (files stored here are available at /dl/file/<repoName>/<buildID>/)
	release/<repoName>/<buildID>/
		<result-filename>
	home/<repoName>/				  (for builds with reused $HOME/uid)
`), dom.br(), docsBuildScript(), dom.h1('Licenses'), dom.p('Ding is open source software. See ', link('licenses', 'licenses'), '.'), dom.h1('Version'), dom.p('This is version ', version, ', ', goversion, ' running on ', goos, '/', goarch, '.'));
	return page;
};
const docsBuildScript = () => {
	return dom.div(dom.h1('Clone'), dom.p('Clones are run as the configured ding user, not under a unique/reused UID. After cloning, file permissions are fixed up. Configure an .ssh/config and/or ssh keys in the home directory of the ding user.'), dom.h1('Build script environment'), dom.p('The build script is run in a clean environment. It should exit with status 0 only when successful. Patterns in the output indicate where build results can be found, such as files and test coverage, see below.'), dom.p('The working directory is set to $DING_BUILDDIR/checkout/$DING_CHECKOUTPATH.'), dom.p('Only a single build will be run for a repository.'), dom.h2('Example'), dom.h3('Basic'), dom.p('Basic build for building ding from github, using the "Build for Go toolchain" setting.'), dom.pre(`#!/usr/bin/env bash
set -eu
export CGO_ENABLED=0
export GOFLAGS="-trimpath -mod=vendor"
go build
go vet
go test -cover
echo version: $(git describe --tag)
echo release: ding linux amd64 $GOTOOLCHAIN ding
`), dom.br(), dom.h3('More elaborate example'), dom.p('This script has comments, and builds release files for multiple architectures, but only for the current Go toolchain version. Assumed to be run with the "Build for Go toolchains" setting.'), dom.pre(`#!/usr/bin/env bash
set -x # Print commands executed.
set -e # Stop executing script when a command fails.
set -u # Fail when using undefined variables.
set -o pipefail # Fail when one of the commands in a pipeline fails.

# Make binaries more standalone, and more likely to work across different OS versions.
export CGO_ENABLED=0
# -mod=vendor requires dependencies to be present in repository.
# -trimpath appears to put fewer new files in the build cache.
export GOFLAGS="-mod=vendor -trimpath"
# Don't allow fetching data (from the proxy).
export GOPROXY=off

# Find name for application.
name=$(basename $PWD)
# Get either a clean tagged name, or one with a commit hash.
version=$(git describe --always)

# Version to be picked up by ding.
echo version: $version

goversion=$(go version | cut -f3 -d' ')

function build() {
	goos=$1
	goarch=$2

	# Build the binary.
	suffix=''
	if test $goos = 'windows'; then
		suffix=.exe
	fi
	GOOS=$goos GOARCH=$goarch go build -o $DING_REPONAME-$version-$goos-$goarch-$GOTOOLCHAIN$suffix

	# Tell ding about a result file.
	echo release: $DING_REPONAME $goos $goarch $GOTOOLCHAIN $DING_REPONAME-$version-$goos-$goarch-$GOTOOLCHAIN$suffix
}

# Test building.
go build -o /dev/null
go vet

# Run tests, and modify output so ding can pick up the coverage result.
go test -shuffle=on -coverprofile cover.out
go tool cover -html=cover.out -o $DING_DOWNLOADDIR/cover.html
echo coverage-report: cover.html

# Build release results for most recent go toolchain, for linux/amd64, linux/386, ...
if test "$DING_GOTOOLCHAIN" = 'go'; then
	build linux amd64
	build linux 386
fi

# Reformat code, require versioned files did not change.
go fmt ./...
git diff --exit-code
`), dom.br(), dom.p('You can include a script like the above in a repository, and call that.'), dom.p('Run a command like ', dom.tt('ding build -goauto ./build.sh'), ' locally to test build scripts. It sets up similar environment variables as during a normal build, and creates target directories. Then it clones the git or hg repository in the working directory to the temporary destination (first parameter) and builds using build.sh, isolated with bwrap. The resulting output is parsed and a summary printed. If that works, the script is likely to work with a regular build in ding too.'), dom.br(), dom.h2('Environment variables'), dom.ul(dom.li("$HOME, an initially empty directory; for repo's with per-build unique UIDs, equal to $DING_BUILDDIR/home, with reused $HOME/uid set to data/home/$DING_REPONAME."), dom.li('$DING_REPONAME, name of the repository'), dom.li('$DING_BRANCH, the branch of the build'), dom.li('$DING_COMMIT, the commit id/hash, empty if not yet known'), dom.li('$DING_BUILDID, the build number, unique over all builds in ding'), dom.li('$DING_BUILDDIR, where all files related to the build are stored, set to data/build/$DING_REPONAME/$DING_BUILDID/'), dom.li('$DING_DOWNLOADDIR, files stored here are available over HTTP at /dl/file/$DING_REPONAME/$DING_BUILDID/...'), dom.li('$DING_CHECKOUTPATH, where files are checked out as configured for the repository, relative to $DING_BUILDDIR/checkout/'), dom.li('$DING_TOOLCHAINDIR, only if configured, the directory where toolchains are stored, like the Go toolchains'), dom.li('any key/value pair from the "environment" object in the ding config file')), dom.p('If "Build for Go toolchains" is used, the following environment variables will also be set, and PATH is adjusted to include the selected Go toolchain:'), dom.ul(dom.li('$DING_GOTOOLCHAIN, with short name go/goprev/gonext'), dom.li('$DING_NEWGOTOOLCHAIN, set when the reason was a newly installed version of the Go toolchain'), dom.li('$GOTOOLCHAIN, set to version of selected Go toolchain, preventing Go from downloading newer Go toolchains')), dom.br(), dom.h2('Output patterns'), dom.p('The standard output of the release script is parsed for lines that can influence the build results. First word is the literal string, the later words are parameters.'), dom.p('Set the version of this build:'), dom.p(dom._class('indent'), dom.tt('version:', ' ', dom.i(dom._class('mono'), 'string'))), dom.p('Add file to build results:'), dom.p(dom._class('indent'), dom.tt('release:', ' ', dom.i(dom._class('mono'), 'command os arch toolchain path'))), dom.ul(dom.li(dom.i('command'), ' is the name of the command, as you would type it in a terminal'), dom.li(dom.i('os'), ' must be one of: ', dom.i('any, linux, darwin, openbsd, windows'), '; the OS this program can run on, ', dom.i('any'), ' is for platform-independent tools like a jar'), dom.li(dom.i('arch'), ' must be one of: ', dom.i('any, amd64, arm64'), '; similar to OS'), dom.li(dom.i('toolchain'), ' should describe the compiler and possibly other tools that are used to build this release'), dom.li(dom.i('path'), ' is the local path (either absolute or relative to the checkout directory) of the released file')), dom.p('Specify test coverage in percentage from 0 to 100 as floating point (an optional trailing "% ..." is ignored):'), dom.p(dom._class('indent'), dom.tt('coverage:', ' ', dom.i(dom._class('mono'), 'float'))), dom.p('Filename (must be relative to $DING_DOWNLOADDIR) for more details about the code coverage, e.g. an html coverage file:'), dom.p(dom._class('indent'), dom.tt('coverage-report:', ' ', dom.i(dom._class('mono'), 'file'))));
};
const pageRepo = async (repoName) => {
	const page = new Page();
	let [repo, builds0, [, mailEnabled, haveGoToolchainDir, settings]] = await authed(() => Promise.all([
		client.Repo(password, repoName),
		client.Builds(password, repoName),
		client.Settings(password),
	]));
	let builds = builds0 || [];
	if (builds.length === 0) {
		setFavicon(favicons.gray);
	}
	else {
		buildSetFavicon(builds[0]);
	}
	const buildsElem = dom.div();
	const atexit = page.newAtexit();
	const renderBuilds = () => {
		atexit.run();
		dom._kids(buildsElem, dom.h1('Builds'), dom.table(dom._class('striped', 'wide'), dom.thead(dom.tr(['ID', 'Branch', 'Status', 'Duration', 'Version', 'Coverage', 'Disk usage', 'Age'].map(s => dom.th(s)), dom.th(style({ textAlign: 'left' }), 'Error'), dom.th('Actions'))), dom.tbody(builds.length === 0 ? dom.tr(dom.td(attr.colspan('10'), 'No builds', style({ textAlign: 'left' }))) : [], builds.map(b => dom.tr(dom.td(link('#repo/' + encodeURIComponent(repo.Name) + '/build/' + b.ID, '' + b.ID)), dom.td(b.Branch), dom.td(buildStatus(b)), dom.td(b.Start ? atexit.age(b.Start, b.Finish || undefined) : ''), dom.td(b.Version, b.CommitHash ? attr.title('Commit ' + b.CommitHash) : []), dom.td(formatCoverage(repo, b)), dom.td(formatBuildSize(b)), dom.td(atexit.ageMins(b.Created, undefined)), dom.td(style({ textAlign: 'left' }), buildErrmsg(b)), dom.td(dom.clickbutton('Rebuild', attr.title('Start new build.'), async function click(e) {
			const nb = await authed(() => client.BuildCreate(password, repo.Name, b.Branch, b.CommitHash, false), e.target);
			if (!builds.find(b => b.ID === nb.ID)) {
				builds.unshift(nb);
				renderBuilds();
			}
		}), ' ', dom.clickbutton('Clear', b.BuilddirRemoved ? attr.disabled('') : [], attr.title('Remove build directory, freeing up disk space.'), async function click(e) {
			await authed(() => client.BuildCleanupBuilddir(password, repo.Name, b.ID), e.target);
			b.BuilddirRemoved = true;
			renderBuilds();
		}), ' ', dom.clickbutton('Remove', b.Released ? attr.disabled('') : [], attr.title('Remove build.'), async function click(e) {
			await authed(() => client.BuildRemove(password, b.ID), e.target);
			builds = builds.filter(xb => xb !== b);
			renderBuilds();
		})))))));
	};
	renderBuilds();
	page.subscribe(streams.build, (e) => {
		if (e.Build.RepoName !== repo.Name) {
			return;
		}
		const i = builds.findIndex(b => b.ID === e.Build.ID);
		if (i < 0) {
			builds.unshift(e.Build);
		}
		else {
			builds[i] = e.Build;
		}
		buildSetFavicon(builds[0]);
		renderBuilds();
	});
	page.subscribe(streams.removeBuild, (e) => {
		if (e.RepoName !== repo.Name) {
			return;
		}
		builds = builds.filter(b => b.ID !== e.BuildID);
		if (builds.length === 0) {
			setFavicon(favicons.gray);
		}
		else {
			buildSetFavicon(builds[0]);
		}
		renderBuilds();
	});
	let name;
	let vcs;
	let origin;
	let originBox;
	let originInput;
	let originTextarea;
	let defaultBranch;
	let checkoutPath;
	let reuseUID;
	let bubblewrap;
	let bubblewrapNoNet;
	let buildOnUpdatedToolchain;
	let goauto;
	let gocur;
	let goprev;
	let gonext;
	let notifyEmailAddrs;
	let buildScript;
	let fieldset;
	const originTextareaBox = dom.div(originTextarea = dom.textarea(repo.Origin, attr.required(''), attr.rows('5'), style({ width: '100%' })), dom.div('Script that clones a repository into checkout/$DING_CHECKOUTPATH.'), dom.div('Typically starts with "#!/bin/sh".'), dom.div('It must print a line of the form "commit: ...".'), dom.br());
	const vcsChanged = function change() {
		if (vcs.value !== 'command') {
			const n = dom.div(originInput);
			originBox.replaceWith(n);
			originBox = n;
			origin = originInput;
		}
		else {
			originBox.replaceWith(originTextareaBox);
			originBox = originTextareaBox;
			origin = originTextarea;
		}
	};
	dom._kids(crumbElem, link('#', 'Home'), ' / ', 'Repo ' + repoName);
	document.title = 'Ding - Repo ' + repoName;
	const render = () => [
		dom.div(style({ marginBottom: '1ex' }), dom.clickbutton('Remove repository', attr.title('Remove repository and all builds, including releases.'), async function click(e) {
			if (!confirm('Are you sure?')) {
				return;
			}
			await authed(() => client.RepoRemove(password, repo.Name), e.target);
			location.hash = '#';
		}), ' ', repo.UID ? dom.clickbutton('Clear home directory', attr.title('Remove shared home directory for this build.'), async function click(e) {
			await authed(() => client.RepoClearHomedir(password, repo.Name), e.target);
		}) : [], ' ', dom.clickbutton('Build', attr.title('Start a build for the default branch of this repository.'), async function click(e) {
			const nb = await authed(() => client.BuildCreate(password, repo.Name, repo.DefaultBranch, '', false), e.target);
			location.hash = '#repo/' + encodeURIComponent(repo.Name) + '/build/' + nb.ID;
		}), ' ', dom.clickbutton('Build ...', attr.title('Create build for specific branch, possibly low-priority.'), async function click() {
			let branch;
			let commit;
			let lowprio;
			const close = popup(dom.h1('New build'), dom.form(async function submit(e) {
				e.stopPropagation();
				e.preventDefault();
				const nb = await authed(() => client.BuildCreate(password, repo.Name, branch.value, commit.value, lowprio.checked), fieldset);
				if (!builds.find(b => b.ID === nb.ID)) {
					builds.unshift(nb);
					renderBuilds();
				}
				close();
			}, dom.fieldset(dom.div(style({ display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top' }), 'Branch', branch = dom.input(attr.required(''), attr.value(repo.DefaultBranch)), dom.div('Commit (optional)', style({ whiteSpace: 'nowrap' })), commit = dom.input(), dom.div(), dom.label(lowprio = dom.input(attr.type('checkbox')), ' Low priority', attr.title('Create build, but only start it when there are no others in progress.'))), dom.br(), dom.submitbutton('Create'))));
			branch.focus();
		})),
		dom.div(style({ display: 'grid', gap: '1em', gridTemplateColumns: '1fr 1fr', justifyItems: 'stretch' }), buildsElem, dom.div(style({ maxWidth: '50em' }), dom.div(dom.h1('Repository settings'), dom.form(async function submit(e) {
			e.stopPropagation();
			e.preventDefault();
			const nr = {
				Name: name.value,
				VCS: vcs.value,
				Origin: origin.value,
				DefaultBranch: defaultBranch.value,
				CheckoutPath: checkoutPath.value,
				UID: !reuseUID.checked ? null : (repo.UID || 1),
				Bubblewrap: bubblewrap.checked,
				BubblewrapNoNet: bubblewrapNoNet.checked,
				BuildOnUpdatedToolchain: buildOnUpdatedToolchain.checked,
				NotifyEmailAddrs: notifyEmailAddrs.value ? notifyEmailAddrs.value.split(',').map(s => s.trim()) : [],
				WebhookSecret: '',
				AllowGlobalWebhookSecrets: false,
				BuildScript: buildScript.value,
				HomeDiskUsage: 0,
				GoAuto: goauto.checked,
				GoCur: gocur.checked,
				GoPrev: goprev.checked,
				GoNext: gonext.checked,
			};
			repo = await authed(() => client.RepoSave(password, nr), fieldset);
		}, fieldset = dom.fieldset(dom.div(style({ display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top' }), 'Name', name = dom.input(attr.disabled(''), attr.value(repo.Name)), dom.span('VCS', attr.title('Clones are run as the configured ding user, not under a unique/reused UID. After cloning, file permissions are fixed up. Configure an .ssh/config and/or ssh keys in the home directory of the ding user.')), vcs = dom.select(dom.option('git', repo.VCS == 'git' ? attr.selected('') : []), dom.option('mercurial', repo.VCS == 'mercurial' ? attr.selected('') : []), dom.option('command', repo.VCS == 'command' ? attr.selected('') : []), vcsChanged), 'Origin', originBox = dom.div(originInput = origin = dom.input(attr.value(repo.Origin), attr.required(''), attr.placeholder('https://... or ssh://... or user@host:path.git'), style({ width: '100%' }))), dom.div('Default branch', style({ whiteSpace: 'nowrap' })), defaultBranch = dom.input(attr.value(repo.DefaultBranch), attr.placeholder('main, master, default')), dom.div('Checkout path', style({ whiteSpace: 'nowrap' })), checkoutPath = dom.input(attr.value(repo.CheckoutPath), attr.required(''), attr.title('Name of the directory to checkout the repository. Go builds may use this name for the binary it creates.')), dom.div('Notify email addresses', style({ whiteSpace: 'nowrap' }), mailEnabled ? [] : [' *', attr.title('No SMTP server is configured for outgoing emails.')]), notifyEmailAddrs = dom.input(attr.value((repo.NotifyEmailAddrs || []).join(', ')), attr.title('Comma-separated list of email address that will receive notifications when a build breaks or is fixed. If empty, the email address configured in the configuration file receives a notification, if any.'), attr.placeholder((settings.NotifyEmailAddrs || []).join(', ') || 'user@example.org, other@example.org')), dom.div(), dom.label(reuseUID = dom.input(attr.type('checkbox'), repo.UID !== null ? attr.checked('') : []), ' Reuse $HOME and UID for builds for this repo', attr.title('By reusing $HOME and running builds for this repository under the same UID, build caches can be used. This typically leads to faster builds but reduces isolation of builds.')), dom.div(), dom.label(bubblewrap = dom.input(attr.type('checkbox'), repo.Bubblewrap ? attr.checked('') : []), ' Run build script in bubblewrap, with limited system access', attr.title('Only available on Linux, with bubblewrap (bwrap) installed. Commands are run in a new mount namespace with access to system directories like /bin /lib /usr, and to the ding build, home and toolchain directories.')), dom.div(), dom.label(bubblewrapNoNet = dom.input(attr.type('checkbox'), repo.BubblewrapNoNet ? attr.checked('') : []), ' Prevent network access from build script. Only active if bubblewrap is active.', attr.title('Hide network interfaces from the build script. Only a loopback device is available.')), dom.div('Build for Go toolchains', style({ whiteSpace: 'nowrap' }), attr.title('The build script will be run for each of the selected Go toolchains. The short name (go, goprev, gonext) is set in $DING_GOTOOLCHAIN. If this build was triggered due to a new Go toolchain being installed, the variable $DING_NEWGOTOOLCHAIN is set.' + !haveGoToolchainDir ? ' Warning: No Go toolchain directory is configured in the configuration file.' : '')), dom.div(dom.label(goauto = dom.input(attr.type('checkbox'), repo.GoAuto ? attr.checked('') : [], function change() {
			if (goauto.checked) {
				gocur.checked = false;
				goprev.checked = false;
				gonext.checked = false;
			}
		}), ' Automatic', attr.title('Build for each of the available Go toolchains, go/goprev/gonext. At least one must be found or the build will fail.')), ' ', dom.label(gocur = dom.input(attr.type('checkbox'), repo.GoCur ? attr.checked('') : [], function change() { goauto.checked = false; }), ' Go latest', attr.title('Latest patch version of latest stable Go toolchain version.')), ' ', dom.label(goprev = dom.input(attr.type('checkbox'), repo.GoPrev ? attr.checked('') : [], function change() { goauto.checked = false; }), ' Go previous', attr.title('Latest patch version of Go toolchain minor version before the latest stable.')), ' ', dom.label(gonext = dom.input(attr.type('checkbox'), repo.GoNext ? attr.checked('') : [], function change() { goauto.checked = false; }), ' Go next', attr.title('Release candidate of Go toolchain, if available.')), ' '), dom.div(), dom.label(buildOnUpdatedToolchain = dom.input(attr.type('checkbox'), repo.BuildOnUpdatedToolchain ? attr.checked('') : []), ' Schedule a low-priority build when new toolchains are automatically installed.')), dom.div(dom.label(dom.div('Build script', style({ marginBottom: '.25ex' })), buildScript = dom.textarea(repo.BuildScript, attr.required(''), attr.rows('24'), style({ width: '100%' })))), dom.br(), dom.div(dom.submitbutton('Save'))))), dom.br(), dom.h1('Webhooks'), dom.p('Configure the following webhook URLs to trigger builds:'), dom.ul(dom.li(dom.tt('http[s]://[webhooklistener]/github/' + repo.Name), ', with secret: ', dom.tt(repo.WebhookSecret)), dom.li(dom.tt('http[s]://[webhooklistener]/gitea/' + repo.Name), ', with secret: ', dom.tt(repo.WebhookSecret)), dom.li(dom.tt('http[s]://[webhooklistener]/bitbucket/' + repo.Name + '/' + repo.WebhookSecret))), repo.AllowGlobalWebhookSecrets && (settings.GithubWebhookSecret || settings.GiteaWebhookSecret || settings.BitbucketWebhookSecret) ? dom.p('Warning: Globally configured webhook secrets are active and also accepted for this repository.') : dom.p('No other (globally configured) secrets are accepted for this repository.'), dom.div(docsBuildScript()), dom.h1('Build settings'), (settings.RunPrefix || []).length > 0 ? dom.p('Build commands are prefixed with: ', dom.tt((settings.RunPrefix || []).join(' '))) : dom.p('Build commands are not run within other commands.'), dom.div('Additional environments available during builds:'), (settings.Environment || []).length === 0 ? dom.p('None') : dom.ul((settings.Environment || []).map(s => dom.li(dom.tt(s)))))),
	];
	const elem = render();
	vcsChanged();
	dom._kids(pageElem, elem);
	return page;
};
const basename = (s) => {
	const t = s.split('/');
	return t[t.length - 1];
};
const pageBuild = async (repoName, buildID) => {
	const page = new Page();
	let [repo, b] = await authed(() => Promise.all([
		client.Repo(password, repoName),
		client.Build(password, repoName, buildID),
	]));
	let steps = b.Steps || [];
	let results = b.Results || [];
	// Builds that were started with this view open. We'll show links to these builds in the top bar.
	let moreBuilds = [];
	let moreBuildsElem = dom.span();
	page.updateRoot = moreBuildsElem;
	const stepColor = () => {
		if (!b.Finish) {
			return colors.gray;
		}
		if (b.Status == api.BuildStatus.StatusSuccess) {
			return colors.green;
		}
		return colors.red;
	};
	dom._kids(crumbElem, dom.span(link('#', 'Home'), ' / ', link('#repo/' + encodeURIComponent(repo.Name), 'Repo ' + repo.Name), ' / ', 'Build ' + b.ID));
	document.title = 'Ding - Repo ' + repoName + ' - Build ' + b.ID;
	buildSetFavicon(b);
	const renderMoreBuilds = () => {
		if (moreBuilds.length === 0) {
			dom._kids(moreBuildsElem);
		}
		else {
			dom._kids(moreBuildsElem, 'New/updated build: ', moreBuilds.map(bID => [link('#repo/' + encodeURIComponent(repo.Name) + '/build/' + bID, '' + bID), ' ']));
		}
	};
	let stepsBox;
	let stepViews;
	const newStepView = (step) => {
		const stepOutput = dom.pre(step.Output, style({ borderLeft: '4px solid ' + stepColor() }));
		const v = {
			output: stepOutput,
			root: dom.div(dom.h2(step.Name, step.Nsec ? ' (' + (step.Nsec / (1000 * 1000 * 1000)).toFixed(3) + 's)' : ''), stepOutput, dom.br())
		};
		return v;
	};
	const atexit = page.newAtexit();
	const render = () => {
		atexit.run();
		dom._kids(pageElem, dom.div(style({ marginBottom: '1ex' }), dom.clickbutton('Remove build', b.Released ? attr.disabled('') : [], attr.title('Remove this build completely from the file system and database.'), async function click(e) {
			await authed(() => client.BuildRemove(password, b.ID), e.target);
			location.hash = '#repo/' + encodeURIComponent(repo.Name);
		}), ' ', dom.clickbutton('Cleanup build dir', attr.title('Remove build directory, freeing up disk spaces.'), b.BuilddirRemoved || !b.Start ? attr.disabled('') : [], async function click(e) {
			b = await authed(() => client.BuildCleanupBuilddir(password, repo.Name, b.ID), e.target);
			render();
		}), ' ', dom.clickbutton('Cancel build', attr.title('Abort this build, causing it to fail.'), b.Finish ? attr.disabled('') : [], async function click(e) {
			await authed(() => client.BuildCancel(password, repo.Name, b.ID), e.target);
		}), ' ', dom.clickbutton('Rebuild', attr.title('Start a new build for this branch and commit.'), async function click(e) {
			const nb = await authed(() => client.BuildCreate(password, repo.Name, b.Branch, b.CommitHash, false), e.target);
			location.hash = '#repo/' + encodeURIComponent(repo.Name) + '/build/' + nb.ID;
		}), ' ', dom.clickbutton('Release', b.Released || b.Status !== api.BuildStatus.StatusSuccess ? attr.disabled('') : [], attr.title("Mark this build as released. Results of releases are not automatically removed. Build directories of releases can otherwise still be automatically removed, but this is done later than for builds that aren't released."), async function click(e) {
			b = await authed(() => client.ReleaseCreate(password, repo.Name, b.ID), e.target);
			render();
		})), dom.div(dom.h1('Summary'), dom.table(dom.tr(['Status', 'Branch', 'Duration', 'Version', 'Commit', 'Coverage', 'Disk usage', 'Age'].map(s => dom.th(s)), dom.th(style({ textAlign: 'left' }), 'Error')), dom.tr(dom.td(buildStatus(b)), dom.td(b.Branch), dom.td(b.Start ? atexit.age(b.Start, b.Finish || undefined) : ''), dom.td(b.Version), dom.td(b.CommitHash), dom.td(formatCoverage(repo, b)), dom.td(formatBuildSize(b)), dom.td(atexit.ageMins(b.Created, undefined)), dom.td(style({ textAlign: 'left' }), b.ErrorMessage ? dom.div(b.ErrorMessage, style({ maxWidth: '40em' })) : [])))), dom.br(), dom.div(style({ display: 'grid', gap: '1em', gridTemplateColumns: '1fr 1fr', justifyItems: 'stretch' }), dom.div(dom.h1('Steps'), stepsBox = dom.div(stepViews = steps.map((step) => newStepView(step)))), dom.div(dom.div(dom.div(style({ display: 'flex', gap: '1em' }), dom.h1('Results'), b.Status === api.BuildStatus.StatusSuccess && (b.Results || []).length > 0 ? dom.div(dom.a(attr.href('dl/' + (b.Released ? 'release' : 'result') + '/' + encodeURIComponent(repo.Name) + '/' + b.ID + '/' + encodeURIComponent(repo.Name) + '-' + b.Version + '.zip'), attr.download(''), 'zip'), ' ', dom.a(attr.href('dl/' + (b.Released ? 'release' : 'result') + '/' + encodeURIComponent(repo.Name) + '/' + b.ID + '/' + encodeURIComponent(repo.Name) + '-' + b.Version + '.tgz'), attr.download(''), 'tgz')) : []), dom.table(dom.thead(dom.tr(['Name', 'OS', 'Arch', 'Toolchain', 'Link', 'Size'].map(s => dom.th(s)))), dom.tbody(results.length === 0 ? dom.tr(dom.td(attr.colspan('6'), 'No results', style({ textAlign: 'left' }))) : [], results.map(rel => dom.tr(dom.td(rel.Command), dom.td(rel.Os), dom.td(rel.Arch), dom.td(rel.Toolchain), dom.td(dom.a(attr.href((b.Released ? 'release/' : 'result/') + encodeURIComponent(repo.Name) + '/' + b.ID + '/' + (b.Released ? basename(rel.Filename) : rel.Filename)), attr.download(''), rel.Filename)), dom.td(formatSize(rel.Filesize))))))), dom.br(), dom.div(dom.h1('Build script'), dom.pre(b.BuildScript)))));
	};
	render();
	page.subscribe(streams.build, (e) => {
		if (e.Build.RepoName !== repo.Name) {
			return;
		}
		if (e.Build.ID === b.ID) {
			b = e.Build;
			results = b.Results || [];
			render();
			buildSetFavicon(b);
		}
		else if (!moreBuilds.includes(e.Build.ID)) {
			moreBuilds.push(e.Build.ID);
			renderMoreBuilds();
		}
	});
	page.subscribe(streams.removeBuild, (e) => {
		if (e.RepoName !== repo.Name || e.BuildID === b.ID) {
			return;
		}
		moreBuilds = moreBuilds.filter(bID => bID !== e.BuildID);
		renderMoreBuilds();
	});
	page.subscribe(streams.output, (e) => {
		if (e.BuildID !== b.ID) {
			return;
		}
		let st = steps.find(st => st.Name === e.Step);
		if (!st) {
			st = {
				Name: e.Step,
				Output: '',
				Nsec: 0,
			};
			for (const sv of stepViews) {
				sv.output.style.borderLeftColor = stepColor();
			}
			steps.push(st);
			const sv = newStepView(st);
			stepViews.push(sv);
			stepsBox.append(sv.root);
		}
		// Scroll new text into view if bottom is already visible.
		const scroll = Math.abs(document.body.getBoundingClientRect().bottom - window.innerHeight) < 50;
		st.Output += e.Text;
		stepViews[stepViews.length - 1].output.innerText += e.Text;
		if (scroll) {
			window.scroll({ top: document.body.scrollHeight });
		}
	});
	return page;
};
let curPage;
const hashchange = async (e) => {
	const hash = decodeURIComponent(window.location.hash.substring(1));
	const t = hash.split('/');
	try {
		let p;
		if (t.length === 1 && t[0] === '') {
			p = await pageHome();
		}
		else if (t.length === 1 && t[0] === 'gotoolchains') {
			p = await pageGoToolchains();
		}
		else if (t.length === 1 && t[0] === 'settings') {
			p = await pageSettings();
		}
		else if (t.length === 1 && t[0] === 'docs') {
			p = await pageDocs();
		}
		else if (t.length === 2 && t[0] === 'repo') {
			p = await pageRepo(t[1]);
		}
		else if (t.length === 4 && t[0] === 'repo' && t[2] === 'build' && parseInt(t[3])) {
			p = await pageBuild(t[1], parseInt(t[3]));
		}
		else {
			window.alert('Unknown hash');
			location.hash = '#';
			return;
		}
		if (curPage) {
			curPage.cleanup();
		}
		curPage = p;
		dom._kids(updateElem, p.updateRoot || []);
	}
	catch (err) {
		window.alert('Error: ' + err.message);
		window.location.hash = e?.oldURL ? new URL(e.oldURL).hash : '';
		throw err;
	}
};
const init = async () => {
	try {
		password = window.localStorage.getItem('dingpassword') || '';
	}
	catch (err) {
		console.log('setting password storage', err);
	}
	if (!password) {
		// Trigger login popup before trying any actual call.
		await authed(async () => {
			if (!password) {
				throw { code: 'user:noAuth', message: 'no session' };
			}
		});
	}
	document.getElementsByTagName('head')[0].append(favicon);
	const root = dom.div(dom.div(style({ display: 'flex', justifyContent: 'space-between', marginBottom: '1ex', padding: '.5em 1em', backgroundColor: '#f8f8f8' }), crumbElem, updateElem, dom.div(sseElem, ' ', link('#docs', 'Docs'), ' ', dom.clickbutton('Logout', function click() {
		try {
			window.localStorage.removeItem('dingpassword');
		}
		catch (err) {
			console.log('remove from session storage', err);
		}
		password = '';
		location.reload();
	}))), dom.div(pageElem));
	document.getElementById('rootElem').replaceWith(root);
	rootElem = root;
	window.addEventListener('hashchange', hashchange);
	await hashchange();
};
window.addEventListener('load', async () => {
	try {
		await init();
	}
	catch (err) {
		window.alert('Error: ' + err.message);
	}
});
