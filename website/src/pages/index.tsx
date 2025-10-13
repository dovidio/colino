import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';

import styles from './index.module.css';

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();

  return (
    <div>
      <div className={styles.centeredContainer}>
        <div className={styles.macWindow}>
          <div className={styles.windowHeader}>
            <div className={styles.trafficLights}>
              <div className={styles.trafficLight + ' ' + styles.red}></div>
              <div className={styles.trafficLight + ' ' + styles.yellow}></div>
              <div className={styles.trafficLight + ' ' + styles.green}></div>
            </div>
          </div>
          <div className={styles.windowContent}>
            <img
              src={require('@site/static/img/banner.gif').default}
              alt="Colino Demo"
              className={styles.windowGif}
            />
          </div>
        </div>
        <div className={styles.buttons}>
          <Link
            className="button button--primary button--lg"
            to="https://github.com/dovidio/colino/releases"
            style={{
              marginRight: '1rem',
              background: 'transparent',
              backgroundColor: 'transparent',
              backgroundImage: 'none',
              color: 'white',
              border: '2px solid white',
              fontFamily: 'Monaco, Menlo, "Ubuntu Mono", Consolas, "Courier New", monospace',
              textTransform: 'uppercase',
              letterSpacing: '0.1em',
              boxShadow: 'none',
              outline: 'none'
            }}
          >
            Download
          </Link>
          <Link
            className="button button--secondary button--lg"
            to="/docs/introduction"
            style={{
              background: 'transparent',
              backgroundColor: 'transparent',
              backgroundImage: 'none',
              color: 'white',
              border: '2px solid white',
              fontFamily: 'Monaco, Menlo, "Ubuntu Mono", Consolas, "Courier New", monospace',
              textTransform: 'uppercase',
              letterSpacing: '0.1em',
              boxShadow: 'none',
              outline: 'none'
            }}
          >
            Docs
          </Link>
        </div>
      </div>
    </div>
  );
}
