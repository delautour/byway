
import * as  React from 'react'
import * as ReactDOM from 'react-dom'
import { Router, router, browserPathDidChange as routerPathDidChange } from 'routelette'
import * as nav from 'navlette'
import * as cn from 'classnames'



nav.onBrowserPathDidChange(routerPathDidChange)


import 'whatwg-fetch'

declare function require(name: string): any;

const style = require('./main.scss')

/* 
Types
*/

type RewriteList = string[]

interface Map<T> {
    [key: string]: T
}

interface BindingConfig {
    scheme: string
    host: string
    rewrite: string
    headers: Map<string>
}

type ServiceMap = Map<Map<BindingConfig>>

interface BywayConfig {
    rewrites: RewriteList
    services: ServiceMap
}

/*
    Commands
*/

function deleteRewrite(index: number, rewrite: string) {
    return () => {
        const data = new FormData()
        fetch("http://localhost:1081/deleteRewrite", {
            method: "POST",
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `index=${index}&rewrite=${encodeURIComponent(rewrite)}`
        })
            .then(refresh)
    }
}

function createRewrite(rewrite: string) {
    return () => {
        const data = new FormData()
        fetch("http://localhost:1081/rewrite", {
            method: "POST",
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `rewrite=${encodeURIComponent(rewrite)}`
        })
            .then(refresh)
    }
}


function createService(name: string) {

    return () => {
        const data = new FormData()
        fetch("http://localhost:1081/createService", {
            method: "POST",
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `name=${encodeURIComponent(name)}`
        })
            .then(refresh)
    }
}


/*
 JSX elements
*/
function Code({children }: { children?: JSX.Element }) {
    return <pre className={style.code}> {children} </pre>
}

class NewRewrite extends React.Component<{}, { match?: string, replace?: string }> {
    constructor() {
        super()
        this.state = { match: '', replace: '' }
    }
    render() {
        return <div className="row">
            <h4> New Rule </h4>
            <input className="col s5" type="text" value={this.state.match} onChange={e => this.setState({ match: (e.target as HTMLInputElement).value })} />
            <span className="col s1  center-align"> → </span>
            <input className="col s5" type="text" value={this.state.replace} onChange={e => this.setState({ replace: (e.target as HTMLInputElement).value })} />

            <div className="col s1">
                <a className="waves-effect waves-light btn" disabled={!this.state.match || !this.state.replace} onClick={() => {
                    if (!this.state.match || !this.state.replace)
                        return
                    const rewrite = `${this.state.match};${this.state.replace}`

                    createRewrite(rewrite)()
                    this.setState({ match: '', replace: '' })
                }
                }> Create Create</a>
            </div>
        </div>
    }
}

function Rewrites({ rewrites }: { rewrites: RewriteList }) {

    return <div className="card-panel">
        <h2>Rewrite Pipeline</h2>
        <ul >
            {rewrites.map((rewrite, index) => {
                const [match, replace] = rewrite.split(';')
                return <li className="row" key={index} onClick={deleteRewrite(index, rewrite)} >
                    <span className="col s5"> <Code>{match}</Code></span>
                    <span className="col s2 center-align ">→</span>
                    <span className="col s5"> <Code>{replace}</Code></span>
                </li>
            }
            )}
        </ul>
        <hr />
        < NewRewrite />
    </div>
}

class CreateService extends React.Component<{}, { name: string }> {
    constructor() {
        super()
        this.state = { name: '' }
    }

    render() {
        return <div className="row">
            <input
                type="text"
                value={this.state.name}
                onChange={(e) => this.setState({ name: (e.target as HTMLInputElement).value })}
                className="col s9" />
            <div className="col s3" > <a  onClick={createService(this.state.name)} className="btn align-center waves-effect waves-light">+</a></div>
        </div>

    }
}

class Services extends React.Component<{ router: Router, services: ServiceMap }, { service: string }> {
    constructor() {
        super()

        this.state = { service: '' }
    }

    componentWillMount() {
        this.servicePathBuilder = this.props.router.register(":name", ({name}: { name: string }) => {
            this.setState({ service: name })
        })
    }

    render() {
        const services = this.props.services;
        const service = this.state.service && this.props.services[this.state.service]
        return (
            <div className="row">
                <div className="col s3">
                    <ul className="collection">
                        {Object.keys(services).map(serviceName => {
                            return <li className={cn("collection-item", { active: serviceName == this.state.service })} key={serviceName} onClick={() => nav.navTo(
                                this.servicePathBuilder({ name: serviceName })
                            )()} > {serviceName} </li>
                        })
                        }

                        <CreateService />
                    </ul>
                </div>

                {service && <div className="col s9">
                    <ul >{
                        Object.keys(service).map(version => {
                            return <li className="card-panel" key={version}><h3>{version}</h3> <Binding config={service[version]} /> </li>

                        })
                    } </ul>
                </div>
                }
            </div>)
    }

    private servicePathBuilder: (any?) => string
}

function Binding({config}: { config: BindingConfig }) {
    return <dl>
        <dt> Scheme </dt> <dd> {config.scheme} </dd>
        <dt> Host </dt> <dd> {config.host} </dd>
        <dt> Headers </dt> <dd> <dl> {
            Object.keys(config.headers || {}).map(header => {
                return [<dt> {header} </dt>, <dl> {config.headers[header]} </dl>]
            })
        }     </dl></dd>

    </dl>
}

class Byway extends React.Component<BywayConfig, { pageContent: (config: BywayConfig) => JSX.Element }> {
    constructor() {
        super()

        this.state = { pageContent: () => null }


        this.servicesPathBuilder = router.register("services", (_, r) =>
            this.setState({ pageContent: (config) => this.services(config, r) })
        )

        this.rewritePathBuilder = router.register("rewrite", (r) => {
            this.setState({
                pageContent: (config) => this.rewrites(config)
            })
        })

        router.register("", (_, r) => {
            var p = this.servicesPathBuilder({})
            nav.navTo(p)()
        }
        )
    }

    services(config, router: Router) {
        return <Services services={this.props.services} router={router} />

    }

    rewrites(config) {
        return <Rewrites rewrites={this.props.rewrites} />
    }

    render() {
        return (<div>
            <nav>
                <div className="nav-wrapper">
                    <a href="#" className="brand-logo right">Byway</a>
                    <ul id="nav-mobile" className="left">
                        <li><a href="#" onClick={(e) => { e.preventDefault(); nav.navTo(this.servicesPathBuilder({}))() } }  >Services</a></li>
                        <li><a href="#" onClick={(e) => { e.preventDefault(); nav.navTo(this.rewritePathBuilder({}))() } }>Rewrite</a></li>

                    </ul>
                </div>
            </nav>

            <div className="row">


                {this.state.pageContent(this.props)}
            </div>
        </div>)
    }

    private servicesPathBuilder: (any?) => string
    private rewritePathBuilder: (any?) => string
}

function refresh() {
    fetch('http://localhost:1081')
        .then(x => x.json())
        .then(config => {
            ReactDOM.render(<Byway {...config} />, document.querySelector('Byway'))
        })
}

ReactDOM.render(<Byway rewrites={[]} services={{}} />, document.querySelector('Byway'))
refresh()

nav.attach()