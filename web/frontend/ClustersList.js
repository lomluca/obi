import React, { Component } from 'react';
import './App.css';

import ClusterItem from './ClusterItem'

    import config from './config'

export default class extends Component {
    constructor(props) {
        super(props);
        this.state = {
            clusters: []
        };

        this.fetchClusters = this.fetchClusters.bind(this);
    }

    async fetchClusters() {
        try {
            const response = await fetch('/api/clusters', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + localStorage.getItem(config.OBI_TOKEN_KEY)
                }
            });
            this.setState({
                clusters: await response.json()
            })
        }
        catch (err) {
            this.setState({
                clusters: []
            })
        }
    }

    async componentWillMount() {
        await this.fetchClusters()
    }

    render() {
        // Create clusters list
        let content = <p>No Clusters</p>;
        if(this.state.clusters.length) {
            content = this.state.clusters.map(cluster =>
                <ClusterItem cluster={cluster}/>
            );
        }

        return (
            <div className="ClustersList">
                {content}
            </div>
        );
    }
}