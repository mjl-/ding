let rootElem: HTMLElement
let crumbElem = dom.span()
let updateElem = dom.span()
let pageElem = dom.div(style({padding: '1em'}))

const client = new api.Client()

const colors = {
	green: '#66ac4c',
	blue: 'rgb(70, 158, 211)',
	red: 'rgb(228, 77, 52)',
	gray: 'rgb(138, 138, 138)',
}

const link = (href: string, anchor: string) => dom.a(attr.href(href), anchor)

interface TargetDisableable {
	target: {
		disabled: boolean
	}
}

class Stream<T> {
	subscribers: ((e: T) => void)[] = []

	send(e: T) {
		this.subscribers.forEach(fn => fn(e))
	}
	subscribe(fn: (e: T) => void): (() => void) {
		this.subscribers.push(fn)
		return () => {
			this.subscribers = this.subscribers.filter(s => s !== fn)
		}
	}
}

const streams = {
	repo: new Stream<api.EventRepo>(),
	removeRepo: new Stream<api.EventRemoveRepo>(),
	build: new Stream<api.EventBuild>(),
	removeBuild: new Stream<api.EventRemoveBuild>(),
	output: new Stream<api.EventOutput>(),
}

let sseElem = dom.span('Disconnected from live updates.') // Shown in UI next to logout button.
let eventSource: EventSource // We initialize it after first success API call.
let allowReconnect = false
const initEventSource = () => {
	// todo: update ui that we are busy connecting
	dom._kids(sseElem, 'Connecting...')
	eventSource = new window.EventSource('events?password='+encodeURIComponent(password))
	eventSource.addEventListener('open', function() {
		allowReconnect = true
		dom._kids(sseElem)
	})
	eventSource.addEventListener('error', function(event: Event) {
		console.log('sse connection error', event)
		if (allowReconnect) {
			allowReconnect = false
			initEventSource()
		} else {
			// todo: on window focus, we could do another reconnect attempt, timethrottled.
			dom._kids(sseElem,
				'Connection error for live updates. ',
				dom.clickbutton('Reconnect', function click() {
					dom._kids(sseElem)
					initEventSource()
				}),
			)
		}
	})
	eventSource.addEventListener('repo', (e: MessageEvent) => streams.repo.send(api.parser.EventRepo(JSON.parse(e.data))))
	eventSource.addEventListener('removeRepo', (e: MessageEvent) => streams.removeRepo.send(api.parser.EventRemoveRepo(JSON.parse(e.data))))
	eventSource.addEventListener('build', (e: MessageEvent) => streams.build.send(api.parser.EventBuild(JSON.parse(e.data))))
	eventSource.addEventListener('removeBuild', (e: MessageEvent) => streams.removeBuild.send(api.parser.EventRemoveBuild(JSON.parse(e.data))))
	eventSource.addEventListener('output', (e: MessageEvent) => streams.output.send(api.parser.EventOutput(JSON.parse(e.data))))
}

// Atexit helps run cleanup code when a page is unloaded. A page has an atexit to
// which functions can be added. Pages that can rerender parts of their contents
// can create a new atexit for a part, register the cleanup function with their
// page (or higher level atexit), and call run to cleanup before rerendering.
class Atexit {
	fns: (() => void)[] = []
	run() {
		for (const fn of this.fns) {
			fn()
		}
		this.fns = []
	}
	add(fn: () => void) {
		this.fns.push(fn)
	}
	age(start: Date, end?: Date) {
		const [elem, close] = age0(false, start, end)
		this.add(close)
		return elem
	}
	ageMins(start: Date, end?: Date) {
		const [elem, close] = age0(true,start, end)
		this.add(close)
		return elem
	}
}

// Page is a loaded page, used to clean up references to event streams and timers.
class Page {
	atexit = new Atexit()
	updateRoot?: HTMLElement // Box holding status about SSE connection.

	newAtexit(): Atexit {
		const atexit = new Atexit()
		this.atexit.add(() => atexit.run())
		return atexit
	}
	cleanup() {
		this.atexit.run()
	}
	subscribe<T>(s: Stream<T>, fn: (e: T) => void) {
		this.atexit.add(s.subscribe(fn))
	}
}

let loginPromise: Promise<void> | undefined
let password = ''

// authed calls fn and awaits the promise it returns. If the promise fails with an
// error object with .code 'user:badAuth', it shows a popup for a password, then
// calls the function again through authed for any password retries.
const authed = async <T>(fn: () => Promise<T>, elem?: {disabled: boolean}): Promise<T> => {
	const overlay = dom.div(style({position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 2, backgroundColor: '#ffffff00'}))
	document.body.append(overlay)
	pageElem.classList.toggle('loading', true)
	if (elem) {
		elem.disabled = true
	}
	const done = () => {
		overlay.remove()
		pageElem.classList.toggle('loading', false)
		if (elem) {
			elem.disabled = false
		}
	}

	try {
		const r = await fn()
		done()
		if (!eventSource) {
			initEventSource()
		}
		return r
	} catch (err: any) {
		done()
		if (err.code !== 'user:noAuth') {
			alert('Error: '+err.message)
		}
		if (err.code === 'user:badAuth' || err.code === 'user:noAuth') {
			if (!loginPromise) {
				let passwordElem: HTMLInputElement
				loginPromise = new Promise((resolve) => {
					const close = popupOpts(true,
						dom.h1('Login'),
						dom.form(
							function submit(e: SubmitEvent) {
								e.stopPropagation()
								e.preventDefault()
								password = passwordElem.value
								try {
									window.sessionStorage.setItem('dingpassword', password)
								} catch (err) {
									console.log('setting session storage', err)
								}
								resolve()
								close()
							},
							dom.fieldset(
								dom.div(
									dom.label(
										dom.div('Password'),
										passwordElem=dom.input(attr.type('password'), attr.required('')),
									),
								),
								dom.br(),
								dom.div(dom.submitbutton('Login')),
							),
						),
					)
				})
				await loginPromise
				loginPromise = undefined
			} else {
				await loginPromise
			}
			return await authed(fn, elem)
		}
		throw err
	}
}

const formatCoverage = (repo: api.Repo, b: api.Build) => {
	const anchor = b.Coverage ? (Math.round(b.Coverage)+'%') : 'report'
	if (b.CoverageReportFile && !b.BuilddirRemoved) {
		return link('dl/file/'+encodeURIComponent(repo.Name)+'/'+b.ID + '/' + b.CoverageReportFile, anchor)
	}
	return anchor === 'report' ? '' : anchor
}

const age0 = (mins: boolean, start: Date, end?: Date | undefined): [HTMLElement, () => void] => {
	const second = 1
	const minute = 60*second
	const hour = 60*minute
	const day = 24*hour
	const week = 7*day
	const year = 365*day
	const periods = [year, week, day, hour, minute, second]
	const suffix = ['y', 'w', 'd', 'h', 'm', 's']

	const elem = dom.span(attr.title(start.toString()))
	let id = 0
	const cleanup = () => {
		if (id) {
			window.clearTimeout(id)
			id = 0
		}
	}

	const set = () => {
		const e = (end || new Date()).getTime()/1000
		let t = e - start.getTime()/1000
		let nextSecs = 0
		let s = ''
		for (let i = 0; i < periods.length; i++) {
			const p = periods[i]
			if (t >= 2*p || i === periods.length-1 || mins && p === minute) {
				if (p == second && t < 10*second) {
					nextSecs = 0.1
					s = t.toFixed(1)+'s'
					break
				}
				const n = Math.round(t/p)
				s = '' + n + suffix[i]
				const prev = Math.floor(t/p)
				nextSecs = Math.ceil((prev+1)*p - t)
				break
			}
		}
		if (!mins && !end) {
			s += '...'
		}
		dom._kids(elem, s)
		// note: Cannot have delays longer than 24.8 days due to storage as 32 bit in
		// browsers. Session is likely closed/reloaded/refreshed before that time anyway.
		return Math.min(nextSecs, 14*24*3600)
	}

	if (end) {
		set()
		return [elem, cleanup]
	}

	const refresh = () => {
		const nextSecs = set()
		id = window.setTimeout(refresh, nextSecs*1000)
	}
	refresh()
	return [elem, cleanup]
}

const formatSize = (size: number) => (size/(1024*1024)).toFixed(1) + 'm'
const formatBuildSize = (b: api.Build) => formatSize(b.DiskUsage) + (b.HomeDiskUsageDelta ? '+'+formatSize(b.HomeDiskUsageDelta) : '')

const statusColor = (b: api.Build) => {
	if (b.ErrorMessage || b.Finish && b.Status !== api.BuildStatus.StatusSuccess) {
		return colors.red
	} else if (b.Released) {
		return colors.blue
	} else if (b.Finish) {
		return colors.green
	} else {
		return colors.gray
	}
}

const buildStatus = (b: api.Build) => {
	let s: string = b.Status
	if (b.Status === api.BuildStatus.StatusNew && b.LowPrio) {
		s += '↓'
	}
	return dom.span(s, style({fontSize: '.9em', color: 'white', backgroundColor: statusColor(b), padding: '0 .2em', borderRadius: '.15em'}))
}

const buildErrmsg = (b: api.Build) => {
	let msg = b.ErrorMessage
	if (b.ErrorMessage && b.LastLine) {
		msg += ', "'+b.LastLine+'"'
	}
	return msg ? dom.span(style({maxWidth: '40em', display: 'inline-block'}), msg) : []
}

const popupOpts = (opaque: boolean, ...kids: ElemArg[]) => {
	const origFocus = document.activeElement
	const close = () => {
		if (!root.parentNode) {
			return
		}
		root.remove()
		if (origFocus && origFocus instanceof HTMLElement && origFocus.parentNode) {
			origFocus.focus()
		}
	}
	let content: HTMLElement
	const root = dom.div(
		style({position: 'fixed', top: 0, right: 0, bottom: 0, left: 0, backgroundColor: opaque ? '#ffffff' : 'rgba(0, 0, 0, 0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: opaque ? 3 : 1}),
		opaque ? [] : [
			function keydown(e: KeyboardEvent) {
				if (e.key === 'Escape') {
					e.stopPropagation()
					close()
				}
			},
			function click(e: MouseEvent) {
				e.stopPropagation()
				close()
			},
		],
		content=dom.div(
			attr.tabindex('0'),
			style({backgroundColor: 'white', borderRadius: '.25em', padding: '1em', boxShadow: '0 0 20px rgba(0, 0, 0, 0.1)', border: '1px solid #ddd', maxWidth: '95vw', overflowX: 'auto', maxHeight: '95vh', overflowY: 'auto'}),
			function click(e: MouseEvent) {
				e.stopPropagation()
			},
			kids,
		)
	)
	document.body.appendChild(root)
	content.focus()
	return close
}

const popup = (...kids: ElemArg[]) => popupOpts(false, ...kids)

const popupRepoAdd = async () => {
	let vcs: HTMLSelectElement
	let origin: HTMLInputElement | HTMLTextAreaElement
	let originBox: HTMLElement
	let originInput: HTMLInputElement
	let originTextarea: HTMLTextAreaElement
	let name: HTMLInputElement
	let defaultBranch: HTMLInputElement
	let reuseUID: HTMLInputElement
	let fieldset: HTMLFieldSetElement

	let branchChanged = false
	let nameChanged = false

	const originTextareaBox = dom.div(
		originTextarea=dom.textarea(attr.required(''), attr.rows('5'), style({width: '100%'})),
		dom.div('Script that clones a repository into checkout/$DING_CHECKOUTPATH.'),
		dom.div('Typically starts with "#!/bin/sh".'),
		dom.div('It must print a line of the form "commit: ...".'),
		dom.br(),
	)

	const vcsChanged = function change() {
		if (!branchChanged) {
			if (vcs.value === 'git') {
				defaultBranch.value = 'main'
			} else if (vcs.value === 'mercurial') {
				defaultBranch.value = 'default'
			} else if (vcs.value === 'command') {
				defaultBranch.value = ''
			}
		}

		if (vcs.value !== 'command') {
			const n = dom.div(originInput)
			originBox.replaceWith(n)
			originBox = n
			origin = originInput
		} else {
			originBox.replaceWith(originTextareaBox)
			originBox = originTextareaBox
			origin = originTextarea
		}
	}

	const close = popup(
		dom.h1('New repository'),
		dom.form(
			async function submit(e: SubmitEvent) {
				e.stopPropagation()
				e.preventDefault()
				const repo: api.Repo = {
					Name: name.value,
					VCS: vcs.value as api.VCS,
					Origin: origin.value,
					DefaultBranch: defaultBranch.value,
					UID: reuseUID.checked ? 1 : null,
					CheckoutPath: name.value,
					BuildScript: '',
					HomeDiskUsage: 0,
				}
				const r = await authed(() => client.RepoCreate(password, repo), fieldset)
				location.hash = '#repo/'+encodeURIComponent(r.Name)
				close()
			},
			fieldset=dom.fieldset(
				dom.div(
					style({display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top'}),
					'VCS',
					vcs=dom.select(
						dom.option('git'),
						dom.option('mercurial'),
						dom.option('command'),
						vcsChanged,
					),
					'Origin',
					originBox=dom.div(originInput=origin=dom.input(attr.required(''), attr.placeholder('https://... or ssh://... or user@host:path.git'), style({width: '100%'}), function keyup() {
						if (nameChanged) {
							return
						}
						let t = origin.value.split('/')
						let s = t[t.length-1] || t[t.length-2] || ''
						s = s.replace(/\.git$/, '')
						name.value = s
					})),
					'Name',
					name=dom.input(attr.required(''), function change() { nameChanged = true }),
					dom.div('Default branch', style({whiteSpace: 'nowrap'})),
					defaultBranch=dom.input(attr.value('main'), attr.placeholder('main, master, default'), function change() { branchChanged = true }),
					dom.div(),
					dom.label(
						reuseUID=dom.input(attr.type('checkbox'), attr.checked('')),
						' Reuse $HOME and UID for builds for this repo',
						attr.title('By reusing $HOME and running builds for this repository under the same UID, build caches can be used. This typically leads to faster builds but reduces isolation of builds.'),
					),
				),
				dom.br(),
				dom.p('The build script can be configured after creating.'),
				dom.div(
					style({textAlign: 'right'}),
					dom.submitbutton('Add'),
				),
			),
		),
	)
	originInput.focus()
}

const pageHome = async (): Promise<Page> => {
	const page = new Page()
	let rbl = await authed(() => client.RepoBuilds(password)) || []

	dom._kids(crumbElem, 'Home')
	document.title = 'Ding - Repos'

	const atexit = page.newAtexit()
	const render = () => {
		atexit.run()

		dom._kids(pageElem,
			dom.div(
				style({marginBottom: '1ex'}),
				dom.a(attr.href('#toolchains'), 'Toolchains'), ' ',
			),
			dom.div(
				style({marginBottom: '1ex'}),
				dom.clickbutton('Add repo', attr.title('Add new repository, to build.'), function click() {
					popupRepoAdd()
				}), ' ',
				dom.clickbutton('Clear homedirs', attr.title('Remove home directories for all repositories that reuse home directories across builds. Cache in such directories can grow over time, consuming quite some disk space.'), async function click(e: MouseEvent & TargetDisableable) {
					if (!confirm('Are you sure?')) {
						return
					}
					await authed(() => client.ClearRepoHomedirs(password), e.target)
				}), ' ',
				dom.clickbutton('Build all lowprio', attr.title('Schedule builds for all repositories, but at low priority.'), async function click(e: MouseEvent & TargetDisableable) {
					await authed(() => client.BuildsCreateLowPrio(password), e.target)
				})
			),
			dom.table(
				dom._class('striped', 'wide'),
				dom.thead(
					dom.tr(
						['Repo', 'Branch', 'Build ID', 'Status', 'Duration', 'Version', 'Coverage', 'Disk usage', 'Home disk usage', 'Age'].map(s => dom.th(s)),
						dom.th(style({textAlign: 'left'}), 'Error'),
					),
				),
				dom.tbody(
					rbl.length === 0 ? dom.tr(dom.td(attr.colspan('10'), 'No repositories', style({textAlign: 'left'}))) : [],
					rbl.map(rb => {
						if ((rb.Builds || []).length === 0) {
							return dom.tr(
								dom.td(dom.a(rb.Repo.Name, attr.href('#repo/'+encodeURIComponent(rb.Repo.Name))))
							)
						}
						return (rb.Builds || []).map((b, i) =>
							dom.tr(
								i === 0 ? dom.td(dom.a(rb.Repo.Name, attr.href('#repo/'+encodeURIComponent(rb.Repo.Name))), attr.rowspan(''+(rb.Builds || []).length)) : [],
								dom.td(b.Branch),
								dom.td(dom.a(''+b.ID, attr.href('#repo/'+encodeURIComponent(rb.Repo.Name)+'/build/'+b.ID))),
								dom.td(buildStatus(b)),
								dom.td(b.Start ? atexit.age(b.Start, b.Finish || undefined) : ''),
								dom.td(b.Version),
								dom.td(formatCoverage(rb.Repo, b)),
								dom.td(formatBuildSize(b)),
								dom.td(rb.Repo.UID ? dom.span(formatSize(rb.Repo.HomeDiskUsage), attr.title('Of reused home directory')) : []),
								dom.td(atexit.ageMins(b.Created, undefined)),
								dom.td(style({textAlign: 'left'}), buildErrmsg(b)),
							)
						)
					}),
				),
			)
		)
	}

	render()

	page.subscribe(streams.build, (e: api.EventBuild) => {
		const rb = rbl.find(rb => rb.Repo.Name === e.RepoName)
		if (!rb) {
			return
		}
		const builds = rb.Builds || []
		const i = builds.findIndex(b => b.ID === e.Build.ID)
		if (i < 0) {
			builds.push(e.Build)
		} else {
			builds[i] = e.Build
		}
		rb.Builds = builds
		render()
	})
	page.subscribe(streams.removeBuild, (e: api.EventRemoveBuild) => {
		const rb = rbl.find(rb => rb.Repo.Name === e.RepoName)
		if (!rb) {
			return
		}
		rb.Builds = (rb.Builds || []).filter(b => b.ID !== e.BuildID)
		render()
	})
	page.subscribe(streams.repo, (ev: api.EventRepo) => {
		console.log('pageHome repo')
		rbl.unshift({Repo: ev.Repo, Builds: []})
		render()
	})
	page.subscribe(streams.removeRepo, (ev: api.EventRemoveRepo) => {
		console.log('pageHome removeRepo')
		rbl = rbl.filter(rb => rb.Repo.Name !== ev.RepoName)
		render()
	})

	return page
}

const pageToolchains = async (): Promise<Page> => {
	const page = new Page()
	const [available0, [installed0, active0]] = await authed(() =>
		Promise.all([
			client.GoToolchainsListReleased(password),
			client.GoToolchainsListInstalled(password),
		])
	)
	let available = available0 || []
	let installed = installed0 || []
	let active = active0 || []

	dom._kids(crumbElem, link('#', 'Home'), ' / ', 'Toolchains')
	document.title = 'Ding - Toolchains'

	const render = () => {
		const groups: string[][] = []
		for (const s of available) {
			const t = s.split('.')
			if (t.length === 1) {
				groups.push([s])
				continue
			}
			const minor = parseInt(t[1])
			const prefix = t[0]+'.'+minor
			if (groups.length > 0 && groups[groups.length-1][0].startsWith(prefix)) {
				groups[groups.length-1].push(s)
			} else {
				groups.push([s])
			}
		}

		let gocur: HTMLSelectElement
		let goprev: HTMLSelectElement

		dom._kids(pageElem,
			dom.p('Go toolchains can easily be installed in the toolchains directory set in the configuration file. Build scripts can add $toolchaindir/<goversion>/bin to their $PATH.'),
			dom.h1('Current and previous Go toolchains'),
			dom.p('The "current" Go toolchain is available through $toolchaindir/go/bin, and the "previous" Go toolchain through $toolchaindir/go-prev/bin.'),
			dom.table(
				dom.tr(
					dom.td('Current'),
					dom.td(
						dom.form(
							async function submit(e: SubmitEvent) {
								e.stopPropagation()
								e.preventDefault()
								await authed(() => client.GoToolchainActivate(password, gocur.value, 'go'))
								active['go'] = gocur.value
								render()
							},
							dom.fieldset(
								gocur=dom.select(
									dom.option('(none)', attr.value('')),
									installed.map(s => dom.option(s, active['go'] === s ? attr.selected('') : [])),
								),
								' ',
								dom.submitbutton('Set', attr.title('Set Go toolchain as "go"')),
							)
						),
					),
				),
				dom.tr(
					dom.td('Previous'),
					dom.td(
						dom.form(
							async function submit(e: SubmitEvent) {
								e.stopPropagation()
								e.preventDefault()
								await authed(() => client.GoToolchainActivate(password, goprev.value, 'go-prev'))
								active['go-prev'] = goprev.value
								render()
							},
							dom.fieldset(
								goprev=dom.select(
									dom.option('(none)', attr.value('')),
									installed.map(s => dom.option(s, active['go-prev'] === s ? attr.selected('') : [])),
								),
								' ',
								dom.submitbutton('Set', attr.title('Set Go toolchain as "go-prev"')),
							)
						),
					),
				),
			),
			dom.br(),

			dom.h1('Released and installed toolchains'),
			dom.div(
				dom.ul(
					style({lineHeight: '1.75'}),
					groups.map(g =>
						dom.li(
							g.map(s =>
								[
									installed.includes(s) ? dom.span(
										s, ' ',
										dom.clickbutton('-', attr.title('Remove toolchain'), async function click(e: MouseEvent & TargetDisableable) {
											await authed(() => client.GoToolchainRemove(password, s), e.target)
											installed = installed.filter(i => i !== s)
											render()
										}),
									) : dom.clickbutton(s, attr.title('Install this toolchain'), async function click(e: MouseEvent & TargetDisableable) {
										await authed(() => client.GoToolchainInstall(password, s, ''), e.target)
										installed.unshift(s)
										render()
									}),
									' ',
								]
							),
						)
					),
				),
			),
		)
	}
	render()
	return page
}

const pageDocs = async (): Promise<Page> => {
	const page = new Page()

	document.title = 'Ding - Docs'
	dom._kids(crumbElem, link('#', 'Home'), ' / Docs')

	dom._kids(pageElem,
		dom.h1('Introduction'),
		dom.p("Ding is a minimalistic build server for internal use. The goal is to make it easy to build software projects in an isolated environment, ensuring it also works on other people's machines. Ding clones a git or mercurial repository, or runs a custom shell script to clone a project, and runs a shell script to build the software. The shell script should output certain lines that ding recognizes, to find build results, test coverage, etc."),

		dom.h1('Notifications'),
		dom.p('Ding can be configured to send a notification email if a repo breaks (failed build) or is repaired again (successful build after previous failure)'),

		dom.h1('Webhooks'),
		dom.p('For each project to build, first configure a repository and a build script. Optionally configure the code repository to call a ding webhook to start a build. For git, this can be done with post-receive shell script in .git/hooks, or through various settings in web apps like gitea, github and bitbucket. For custom scripts, run ', dom.tt('ding kick baseURL repoName branch commit < password-file'), ' to start a build, where baseURL could be http://localhost:6084 (for default settings), and password is what you use for logging in. For externally-defined webhook formats, ensure the ding webhook listener is publicly accessible (e.g. through a reverse proxy), and configure these paths for the respective services: ', dom.tt('https://.../gitea/<repo>'), ', ', dom.tt('https://.../github/<repo>'), ' or ', dom.tt('https://.../bitbucket/<repo>/<secret>'), '. Gitea includes a "secret" in an Authorization header, github signs its request payload, for bitbucket you must include a secret value in the URL they send the webhook too. These secrets must be configured in the ding configuration file.'),

		dom.h1('Authentication'),
		dom.p('Ding only has simple password-based authentication, with a single password for the entire system. Everyone with the password can see all repositories, builds and scripts, and modify all data.'),

		dom.h1('Go toolchains'),
		dom.p('Ding has builtin functionality for downloading Go toolchains for use in builds.'),

		dom.h1('API'),
		dom.p('Ding has a simple HTTP/JSON-based API, see ', link('ding/', 'Ding API'), '.'),

		dom.h1('Files and directories'),
		dom.p('Ding stores all files for repositories, builds, releases and home directories in its "data" directory:'),
		dom.pre(`
data/
    build/<repoName>/<buildID>/       ($DING_BUILDDIR during builds)
        checkout/$DING_CHECKOUTPATH/  (working directory for build.sh)
        scripts/
            build.sh                  (copied from database before build)
        output/
            {clone,build}.{stdout,stderr,output,nsec}
        home/                         (for builds with unique $HOME/uid)
        dl/                           (files stored here are available at /dl/file/<repoName>/<buildID>/)
    release/<repoName>/<buildID>/
        <result-filename>
    home/<repoName>/                  (for builds with reused $HOME/uid)
`),
		dom.br(),

		docsBuildScript(),
	)

	return page
}

const docsBuildScript = (): HTMLElement => {
	return dom.div(
		dom.h1('Build script environment'),
		dom.p('The build script is run in a clean environment. It should exit with status 0 only when successful. Patterns in the output indicate where build results can be found, such as files and test coverage.'),
		dom.p('The working directory is set to $DING_BUILDDIR/checkout/$DING_CHECKOUTPATH.'),

		dom.h2('Example'),
		dom.pre(`#!/bin/bash
set -xeuo pipefail

export GOOS=linux
export GOARCH=amd64
version=$(git describe --always)
goversion=$(go version | cut -f3 -d' ')

echo version: $version

go build -o app-$version-$GOOS-$GOARCH
go vet
go test -coverprofile cover.out | sed "s/^coverage: \(.*\)% of statements/coverage: \\1/"
go tool cover -html=cover.out -o $DING_DOWNLOADDIR/cover.html
echo coverage-report: cover.html

echo release: app $GOOS $GOARCH $goversion app-$version-$GOOS-$GOARCH`
		),

		dom.br(),
		dom.h2('Environment variables'),
		dom.ul(
			dom.li("$HOME, an initially empty directory; for repo's with per-build unique UIDs, equal to $DING_BUILDDIR/home, with reused $HOME/uid set to data/home/$DING_REPONAME."),
			dom.li('$DING_REPONAME, name of the repository'),
			dom.li('$DING_BRANCH, the branch of the build'),
			dom.li('$DING_COMMIT, the commit id/hash, empty if not yet known'),
			dom.li('$DING_BUILDID, the build number, unique over all builds in ding'),
			dom.li('$DING_BUILDDIR, where all files related to the build are stored, set to data/build/$DING_REPONAME/$DING_BUILDID/'),
			dom.li('$DING_DOWNLOADDIR, files stored here are available over HTTP at /dl/file/$DING_REPONAME/$DING_BUILDID/...'),
			dom.li('$DING_CHECKOUTPATH, where files are checked out as configured for the repository, relative to $DING_BUILDDIR/checkout/'),
			dom.li('any key/value pair from the "environment" object in the ding config file'),
		),

		dom.br(),
		dom.h2('Output patterns'),
		dom.p('The standard output of the release script is parsed for lines that can influence the build results. First word is the literal string, the later words are parameters.'),

		dom.p('Set the version of this build:'),
		dom.p(dom._class('indent'), dom.tt('version:', ' ', dom.i(dom._class('mono'), 'string'))),

		dom.p('Add file to build results:'),
		dom.p(dom._class('indent'), dom.tt('release:', ' ', dom.i(dom._class('mono'), 'command os arch toolchain path'))),
		dom.ul(
			dom.li(dom.i('command'), ' is the name of the command, as you would type it in a terminal'),
			dom.li(dom.i('os'), ' must be one of: ', dom.i('any, linux, darwin, openbsd, windows'), '; the OS this program can run on, ', dom.i('any'), ' is for platform-independent tools like a jar'),
			dom.li(dom.i('arch'), ' must be one of: ', dom.i('any, amd64, arm64'), '; similar to OS'),
			dom.li(dom.i('toolchain'), ' should describe the compiler and possibly other tools that are used to build this release'),
			dom.li(dom.i('path'), ' is the local path (either absolute or relative to the checkout directory) of the released file'),
		),

		dom.p('Specify test coverage in percentage from 0 to 100 as floating point:'),
		dom.p(dom._class('indent'), dom.tt('coverage:', ' ', dom.i(dom._class('mono'), 'float'))),

		dom.p('Filename (must be relative to $DING_DOWNLOADDIR) for more details about the code coverage, e.g. an html coverage file:'),
		dom.p(dom._class('indent'), dom.tt('coverage-report:', ' ', dom.i(dom._class('mono'), 'file'))),
	)
}

const pageRepo = async (repoName: string): Promise<Page> => {
	const page = new Page()
	let [repo, builds0] = await authed(() =>
		Promise.all([
			client.Repo(password, repoName),
			client.Builds(password, repoName)
		])
	)
	let builds = builds0 || []

	const buildsElem = dom.div()

	const atexit = page.newAtexit()
	const renderBuilds = () => {
		atexit.run()

		dom._kids(buildsElem,
			dom.h1('Builds'),
			dom.table(
				dom._class('striped', 'wide'),
				dom.thead(
					dom.tr(
						['ID', 'Branch', 'Status', 'Duration', 'Version', 'Coverage', 'Disk usage', 'Age'].map(s => dom.th(s)),
						dom.th(style({textAlign: 'left'}), 'Error'),
						dom.th('Actions'),
					),
				),
				dom.tbody(
					builds.length === 0 ? dom.tr(dom.td(attr.colspan('10'), 'No builds', style({textAlign: 'left'}))) : [],
					builds.map(b =>
						dom.tr(
							dom.td(dom.a(''+b.ID, attr.href('#repo/'+encodeURIComponent(repo.Name)+'/build/'+b.ID))),
							dom.td(b.Branch),
							dom.td(buildStatus(b)),
							dom.td(b.Start ? atexit.age(b.Start, b.Finish || undefined) : ''),
							dom.td(b.Version),
							dom.td(formatCoverage(repo, b)),
							dom.td(formatBuildSize(b)),
							dom.td(atexit.ageMins(b.Created, undefined)),
							dom.td(style({textAlign: 'left'}), buildErrmsg(b)),
							dom.td(
								dom.clickbutton('Rebuild', attr.title('Start new build.'), async function click(e: TargetDisableable) {
									const nb = await authed(() => client.BuildCreate(password, repo.Name, b.Branch, b.CommitHash, false), e.target)
									if (!builds.find(b => b.ID === nb.ID)) {
										builds.unshift(nb)
										renderBuilds()
									}
								}), ' ',
								dom.clickbutton('Clear', b.BuilddirRemoved ? attr.disabled('') : [], attr.title('Remove build directory, freeing up disk space.'), async function click(e: TargetDisableable) {
									await authed(() => client.BuildCleanupBuilddir(password, repo.Name, b.ID), e.target)
									b.BuilddirRemoved = true
									renderBuilds()
								}), ' ',
								dom.clickbutton('Remove', b.Released ? attr.disabled('') : [], attr.title('Remove build.'), async function click(e: TargetDisableable) {
									await authed(() => client.BuildRemove(password, b.ID), e.target)
									builds = builds.filter(xb => xb !== b)
									renderBuilds()
								}),
							),
						)
					),
				),
			),
		)
	}
	renderBuilds()

	page.subscribe(streams.build, (e: api.EventBuild) => {
		if (e.RepoName !== repo.Name) {
			return
		}
		const i = builds.findIndex(b => b.ID === e.Build.ID)
		if (i < 0) {
			builds.unshift(e.Build)
		} else {
			builds[i] = e.Build
		}
		renderBuilds()
	})
	page.subscribe(streams.removeBuild, (e: api.EventRemoveBuild) => {
		if (e.RepoName !== repo.Name) {
			return
		}
		builds = builds.filter(b => b.ID !== e.BuildID)
		renderBuilds()
	})

	let name: HTMLInputElement
	let vcs: HTMLSelectElement
	let origin: HTMLInputElement | HTMLTextAreaElement
	let originBox: HTMLElement
	let originInput: HTMLInputElement
	let originTextarea: HTMLTextAreaElement
	let defaultBranch: HTMLInputElement
	let checkoutPath: HTMLInputElement
	let reuseUID: HTMLInputElement
	let buildScript: HTMLTextAreaElement
	let fieldset: HTMLFieldSetElement

	const originTextareaBox = dom.div(
		originTextarea=dom.textarea(repo.Origin, attr.required(''), attr.rows('5'), style({width: '100%'})),
		dom.div('Script that clones a repository into checkout/$DING_CHECKOUTPATH.'),
		dom.div('Typically starts with "#!/bin/sh".'),
		dom.div('It must print a line of the form "commit: ...".'),
		dom.br(),
	)

	const vcsChanged = function change() {
		if (vcs.value !== 'command') {
			const n = dom.div(originInput)
			originBox.replaceWith(n)
			originBox = n
			origin = originInput
		} else {
			originBox.replaceWith(originTextareaBox)
			originBox = originTextareaBox
			origin = originTextarea
		}
	}

	dom._kids(crumbElem, link('#', 'Home'), ' / ', 'Repo '+repoName)
	document.title = 'Ding - Repo '+repoName

	const render = () => [
		dom.div(
			style({marginBottom: '1ex'}),
			dom.clickbutton('Remove repository', attr.title('Remove repository and all builds, including releases.'), async function click(e: TargetDisableable) {
				if (!confirm('Are you sure?')) {
					return
				}
				await authed(() => client.RepoRemove(password, repo.Name), e.target)
				location.hash = '#'
			}), ' ',
			repo.UID ? dom.clickbutton('Clear home directory', attr.title('Remove shared home directory for this build.'), async function click(e: TargetDisableable) {
				await authed(() => client.RepoClearHomedir(password, repo.Name), e.target)
			}) : [], ' ',
			dom.clickbutton('Build', attr.title('Start a build for the default branch of this repository.'), async function click(e: TargetDisableable) {
				const nb = await authed(() => client.BuildCreate(password, repo.Name, repo.DefaultBranch, '', false), e.target)
				location.hash = '#repo/'+encodeURIComponent(repo.Name)+'/build/'+nb.ID
			}), ' ',
			dom.clickbutton('Build ...', attr.title('Create build for specific branch, possibly low-priority.'), async function click() {
				let branch: HTMLInputElement
				let lowprio: HTMLInputElement

				const close = popup(
					dom.h1('New build'),
					dom.form(
						async function submit(e: SubmitEvent) {
							e.stopPropagation()
							e.preventDefault()
							const nb = await authed(() => client.BuildCreate(password, repo.Name, branch.value, '', lowprio.checked), fieldset)
							if (!builds.find(b => b.ID === nb.ID)) {
								builds.unshift(nb)
								renderBuilds()
							}
							close()
						},
						dom.fieldset(
							dom.div(
								style({display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top'}),
								'Branch',
								branch=dom.input(attr.required(''), attr.value(repo.DefaultBranch)),
								dom.div(),
								dom.label(
									lowprio=dom.input(attr.type('checkbox')),
									' Low priority',
									attr.title('Create build, but only start it when there are no others in progress.'),
								),
							),
							dom.br(),
							dom.submitbutton('Create'),
						)
					),
				)
				branch.focus()
			}),
		),
		dom.div(
			style({display: 'grid', gap: '1em', gridTemplateColumns: '1fr 1fr', justifyItems: 'stretch'}),
			buildsElem,
			dom.div(
				style({maxWidth: '50em'}),
				dom.div(
					dom.h1('Repository settings'),
					dom.form(
						async function submit(e: SubmitEvent) {
							e.stopPropagation()	
							e.preventDefault()
							const nr: api.Repo = {
								Name: name.value,
								VCS: vcs.value as api.VCS,
								Origin: origin.value,
								DefaultBranch: defaultBranch.value,
								CheckoutPath: checkoutPath.value,
								UID: !reuseUID.checked ? null : (repo.UID || 1),
								BuildScript: buildScript.value,
								HomeDiskUsage: 0,
							}
							repo = await authed(() => client.RepoSave(password, nr), fieldset)
						},
						fieldset=dom.fieldset(
							dom.div(
								style({display: 'grid', columnGap: '1em', rowGap: '.5ex', gridTemplateColumns: 'min-content 1fr', alignItems: 'top'}),
								'Name',
								name=dom.input(attr.disabled(''), attr.value(repo.Name)),
								'VCS',
								vcs=dom.select(
									dom.option('git', repo.VCS == 'git' ? attr.selected('') : []),
									dom.option('mercurial', repo.VCS == 'mercurial' ? attr.selected('') : []),
									dom.option('command', repo.VCS == 'command' ? attr.selected('') : []),
									vcsChanged,
								),
								'Origin',
								originBox=dom.div(originInput=origin=dom.input(attr.value(repo.Origin), attr.required(''), attr.placeholder('https://... or ssh://... or user@host:path.git'), style({width: '100%'}))),
								dom.div('Default branch', style({whiteSpace: 'nowrap'})),
								defaultBranch=dom.input(attr.value(repo.DefaultBranch), attr.placeholder('main, master, default')),
								dom.div('Checkout path', style({whiteSpace: 'nowrap'})),
								checkoutPath=dom.input(attr.value(repo.CheckoutPath), attr.required(''), attr.title('Name of the directory to checkout the repository. Go builds may use this name for the binary it creates.')),
								dom.div(),
								dom.label(
									reuseUID=dom.input(attr.type('checkbox'), repo.UID !== null ? attr.checked('') : []),
									' Reuse $HOME and UID for builds for this repo',
									attr.title('By reusing $HOME and running builds for this repository under the same UID, build caches can be used. This typically leads to faster builds but reduces isolation of builds.'),
								),
							),
							dom.div(
								dom.label(
									dom.div('Build script', style({marginBottom: '.25ex'})),
									buildScript=dom.textarea(repo.BuildScript, attr.required(''), attr.rows('24'), style({width: '100%'})),
								),
							),
							dom.br(),
							dom.div(
								dom.submitbutton('Save')
							),
						),
					),
				),
				dom.br(),
				dom.div(
					docsBuildScript()
				),
			),
		),
	]

	const elem = render()
	vcsChanged()
	dom._kids(pageElem, elem)

	return page
}

const basename = (s: string) => {
	const t = s.split('/')
	return t[t.length-1]
}

const pageBuild = async (repoName: string, buildID: number): Promise<Page> => {
	const page = new Page()
	let [repo, b] = await authed(() =>
		Promise.all([
			client.Repo(password, repoName),
			client.Build(password, repoName, buildID),
		])
	)
	let steps = b.Steps || []
	let results = b.Results || []

	// Builds that were started with this view open. We'll show links to these builds in the top bar.
	let moreBuilds: number[] = []
	let moreBuildsElem = dom.span()
	page.updateRoot = moreBuildsElem

	const stepColor = () => {
		if (!b.Finish) {
			return colors.gray
		}
		if (b.Status == api.BuildStatus.StatusSuccess) {
			return colors.green
		}
		return colors.red
	}

	dom._kids(crumbElem,
		dom.span(link('#', 'Home'), ' / ', link('#repo/'+encodeURIComponent(repo.Name), 'Repo '+repo.Name), ' / ', 'Build '+b.ID),
	)
	document.title = 'Ding - Repo '+repoName + ' - Build '+b.ID

	const renderMoreBuilds = () => {
		if (moreBuilds.length === 0) {
			dom._kids(moreBuildsElem)
		} else {
			dom._kids(moreBuildsElem, 'New/updated build: ', moreBuilds.map(bID => [link('#repo/'+encodeURIComponent(repo.Name)+'/build/'+bID, ''+bID), ' ']))
		}
	}

	let stepsBox: HTMLElement
	let stepViews: StepView[]
	interface StepView {
		root: HTMLElement
		output: HTMLElement
	}
	const newStepView = (step: api.Step) => {
		const stepOutput = dom.pre(step.Output, style({borderLeft: '4px solid '+stepColor()}))
		const v: StepView = {
			output: stepOutput,
			root: dom.div(
				dom.h2(step.Name, step.Nsec ? ' (' + (step.Nsec/(1000*1000*1000)).toFixed(3)+'s)' : ''),
				stepOutput,
				dom.br(),
			)
		}
		return v
	}

	const atexit = page.newAtexit()
	const render = () => {
		atexit.run()

		dom._kids(pageElem,
			dom.div(
				style({marginBottom: '1ex'}),
				dom.clickbutton('Remove build', b.Released ? attr.disabled('') : [], attr.title('Remove this build completely from the file system and database.'), async function click(e: TargetDisableable) {
					await authed(() => client.BuildRemove(password, b.ID), e.target)
					location.hash = '#repo/'+encodeURIComponent(repo.Name)
				}), ' ',
				dom.clickbutton('Cleanup build dir', attr.title('Remove build directory, freeing up disk spaces.'), b.BuilddirRemoved || !b.Start ? attr.disabled('') : [], async function click(e: TargetDisableable) {
					b = await authed(() => client.BuildCleanupBuilddir(password, repo.Name, b.ID), e.target)
					render()
				}), ' ',
				dom.clickbutton('Cancel build', attr.title('Abort this build, causing it to fail.'), b.Finish ? attr.disabled('') : [], async function click(e: TargetDisableable) {
					await authed(() => client.BuildCancel(password, repo.Name, b.ID), e.target)
				}), ' ',
				dom.clickbutton('Rebuild', attr.title('Start a new build for this branch and commit.'), async function click(e: TargetDisableable) {
					const nb = await authed(() => client.BuildCreate(password, repo.Name, b.Branch, b.CommitHash, false), e.target)
					location.hash = '#repo/'+encodeURIComponent(repo.Name)+'/build/'+nb.ID
				}), ' ',
				dom.clickbutton('Release', b.Released || b.Status !== api.BuildStatus.StatusSuccess ? attr.disabled('') : [], attr.title("Mark this build as released. Results of releases are not automatically removed. Build directories of releases can otherwise still be automatically removed, but this is done later than for builds that aren't released."), async function click(e: TargetDisableable) {
					b = await authed(() => client.ReleaseCreate(password, repo.Name, b.ID), e.target)
					render()
				}),
			),
			dom.div(
				dom.h1('Summary'),
				dom.table(
					dom.tr(
						['Status', 'Branch', 'Duration', 'Commit', 'Version', 'Coverage', 'Size', 'Age'].map(s => dom.th(s)),
						dom.th(style({textAlign: 'left'}), 'Error'),
					),
					dom.tr(
						dom.td(buildStatus(b)),
						dom.td(b.Branch),
						dom.td(b.Start ? atexit.age(b.Start, b.Finish || undefined) : ''),
						dom.td(b.CommitHash),
						dom.td(b.Version),
						dom.td(formatCoverage(repo, b)),
						dom.td(formatBuildSize(b)),
						dom.td(atexit.ageMins(b.Created, undefined)),
						dom.td(style({textAlign: 'left'}), b.ErrorMessage ? dom.div(b.ErrorMessage, style({maxWidth: '40em'})) : []),
					),
				),
			),
			dom.br(),
			dom.div(
				style({display: 'grid', gap: '1em', gridTemplateColumns: '1fr 1fr', justifyItems: 'stretch'}),
				dom.div(
					dom.h1('Steps'),
					stepsBox=dom.div(
						stepViews=steps.map((step) => newStepView(step))
					),
				),
				dom.div(
					dom.div(
						dom.div(
							style({display: 'flex', gap: '1em'}),
							dom.h1('Results'),
							b.Status === api.BuildStatus.StatusSuccess ? dom.div(
								link('dl/' + (b.Released ? 'release' : 'result') + '/'+encodeURIComponent(repo.Name) + '/' + b.ID + '/' + encodeURIComponent(repo.Name) + '-' + b.Version + '.zip', 'zip'),' ',
								link('dl/' + (b.Released ? 'release' : 'result') + '/'+encodeURIComponent(repo.Name) + '/' + b.ID + '/' + encodeURIComponent(repo.Name) + '-' + b.Version + '.tgz', 'tgz'),
							) : [],
						),
						dom.table(
							dom.thead(
								dom.tr(
									['Name', 'OS', 'Arch', 'Toolchain', 'Link', 'Size'].map(s => dom.th(s)),
								),
							),
							dom.tbody(
								results.length === 0 ? dom.tr(dom.td(attr.colspan('6'), 'No results', style({textAlign: 'left'}))) : [],
								results.map(rel =>
									dom.tr(
										dom.td(rel.Command),
										dom.td(rel.Os),
										dom.td(rel.Arch),
										dom.td(rel.Toolchain),
										dom.td(link((b.Released ? 'release/' : 'result/') + encodeURIComponent(repo.Name) + '/' + b.ID + '/' + (b.Released ? basename(rel.Filename) : rel.Filename), rel.Filename)),
										dom.td(formatSize(rel.Filesize)),
									)
								),
							),
						),
					),
					dom.br(),
					dom.div(
						dom.h1('Build script'),
						dom.pre(b.BuildScript),
					),
				),
			),
		)
	}
	render()

	page.subscribe(streams.build, (e: api.EventBuild) => {
		if (e.RepoName !== repo.Name) {
			return
		}
		if (e.Build.ID === b.ID) {
			b = e.Build
			results = b.Results || []
			render()
		} else if (!moreBuilds.includes(e.Build.ID)) {
			moreBuilds.push(e.Build.ID)
			renderMoreBuilds()
		}
	})
	page.subscribe(streams.removeBuild, (e: api.EventRemoveBuild) => {
		if (e.RepoName !== repo.Name || e.BuildID === b.ID) {
			return
		}
		moreBuilds = moreBuilds.filter(bID => bID !== e.BuildID)
		renderMoreBuilds()
	})
	page.subscribe(streams.output, (e: api.EventOutput) => {
		if (e.BuildID !== b.ID) {
			return
		}
		let st = steps.find(st => st.Name === e.Step)
		if (!st) {
			st = {
				Name: e.Step as api.BuildStatus,
				Output: '',
				Nsec: 0,
			}
			for (const sv of stepViews) {
				sv.output.style.borderLeftColor = stepColor()
			}
			steps.push(st)
			const sv = newStepView(st)
			stepViews.push(sv)
			stepsBox.append(sv.root)
		}
		st.Output += e.Text
		stepViews[stepViews.length-1].output.innerText += e.Text
	})

	return page
}

let curPage: Page

const hashchange = async (e?: HashChangeEvent) => {
	const hash = decodeURIComponent(window.location.hash.substring(1))
	const t = hash.split('/')

	try {
		let p: Page
		if (t.length === 1 && t[0] === '') {
			p = await pageHome()
		} else if (t.length === 1 && t[0] === 'toolchains') {
			p = await pageToolchains()
		} else if (t.length === 1 && t[0] === 'docs') {
			p = await pageDocs()
		} else if (t.length === 2 && t[0] === 'repo') {
			p = await pageRepo(t[1])
		} else if (t.length === 4 && t[0] === 'repo' && t[2] === 'build' && parseInt(t[3])) {
			p = await pageBuild(t[1], parseInt(t[3]))
		} else {
			window.alert('Unknown hash')
			location.hash = '#'
			return
		}
		if (curPage) {
			curPage.cleanup()
		}
		curPage = p
		dom._kids(updateElem, p.updateRoot || [])
	} catch (err: any) {
		window.alert('Error: '+err.message)
		window.location.hash = e?.oldURL ? new URL(e.oldURL).hash : ''
		throw err
	}
}

const init = async () => {
	try {
		password = window.sessionStorage.getItem('dingpassword') || ''
	} catch(err: any) {
		console.log('setting password storage', err)
	}
	if (!password) {
		// Trigger login popup before trying any actual call.
		await authed(async () => {
			if (!password) {
				throw {code: 'user:noAuth', message: 'no session'}
			}
		})
	}

	const root = dom.div(
		dom.div(
			style({display: 'flex', justifyContent: 'space-between', marginBottom: '1ex', padding: '.5em 1em', backgroundColor: '#f8f8f8'}),
			crumbElem,
			updateElem,
			dom.div(
				sseElem, ' ',
				link('#docs', 'Docs'), ' ',
				dom.clickbutton('Logout', function click() {
					try {
						window.sessionStorage.removeItem('dingpassword')
					} catch (err) {
						console.log('remove from session storage', err)
					}
					password = ''
					location.reload()
				}),
			),
		),
		dom.div(
			pageElem,
		),
	)
	document.getElementById('rootElem')!.replaceWith(root)
	rootElem = root
	window.addEventListener('hashchange', hashchange)
	await hashchange()
}

window.addEventListener('load', async () => {
	try {
		await init()
	} catch (err: any) {
		window.alert('Error: ' + err.message)
	}
})