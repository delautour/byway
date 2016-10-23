
import * as  React from 'react'
import * as ReactDOM from 'react-dom'
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
            body: `index=${index}&rewrite=${encodeURIComponent(rewrite)}` })
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
            body: `rewrite=${encodeURIComponent(rewrite)}` })
        .then(refresh)
    }
}


/*
 JSX elements
*/
function Code({children }: { children?: JSX.Element }) {
    return <pre className={style.code}> {children} </pre>
}

function Rewrites({ rewrites }: { rewrites: RewriteList }) {

    return <div className="card-panel">
    <h2>Rewrite Pipeline</h2> 
    <ul >
        {rewrites.map((rewrite,index) => {
            const [match, replace] = rewrite.split(';')
            return <li key={index} onClick={deleteRewrite(index, rewrite)} > <Code>{match}</Code> → <Code>{replace}</Code> </li>
        }
        )}
    </ul>
    <button onClick={createRewrite("route;to")}> Create a new rewrite</button>
    </div>
}

function Services({services}: { services: ServiceMap }) {
    return <ul>
        { Object.keys(services).map(serviceName => {
            const versions = services[serviceName]
            return <li className="card-panel" key={serviceName}> <h2>{serviceName}</h2> <ul>{

                Object.keys(versions).map(version => {
                    return <li key={version}><h3>{version}</h3> <Binding config={versions[version]}/> </li>

                } )
                } </ul></li>
        })}
    </ul>
}

function Binding ({config}: {config: BindingConfig}){
    return <dl>
        <dt> Scheme </dt> <dd> {config.scheme} </dd>
        <dt> Host </dt> <dd> {config.host} </dd>
        <dt> Headers </dt> <dd> <dl> {
            Object.keys(config.headers || {}).map(header => {
                return [<dt> { header } </dt>, <dl> { config.headers[header] } </dl>]
            })
        }     </dl></dd>

    </dl>
}

function Byway({config}: { config: BywayConfig }) {
    return (
        <div>
            <h1> Byway </h1>
            <Rewrites rewrites={config.rewrites} />
            <hr />
            <Services services={config.services} />
        </div>)
}

function refresh(){
    fetch('http://localhost:1081')
    .then(x => x.json())
    .then(config => {
        ReactDOM.render(<Byway config={config} />, document.querySelector('Byway'))
    })
 }



ReactDOM.render(<Byway config={{ rewrites: [], services: {} }} />, document.querySelector('Byway'))
refresh()