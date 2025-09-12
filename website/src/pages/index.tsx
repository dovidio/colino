import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';

import styles from './index.module.css';



function HomepageHeader() {
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container" style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap'}}>
        <div style={{flex: 1, minWidth: 260}}>
          <Heading as="h1" className="hero__title">
            Colino
          </Heading>
          <p className="hero__subtitle">
            Minimal, privacy-first content aggregation and summarization. Stay informed, not overwhelmed.
          </p>
        </div>
        <img
          src={require('@site/static/img/filtering.png').default}
          alt="Filtering"
          style={{maxWidth: 320, width: '100%', marginLeft: 32, borderRadius: 12}}
        />
      </div>
    </header>
  );
}


export default function Home(): ReactNode {
  return (
    <Layout>
      <HomepageHeader />
      <HomepageFeatures />
    </Layout>
  );
}
