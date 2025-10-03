import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg?: React.ComponentType<React.ComponentProps<'svg'>>;
  imgSrc?: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Privacy Control',
    imgSrc: require('@site/static/img/Privacy.png').default,
    description: (
      <>
        No accounts, no tracking, no cloud services. Your personal knowledge garden stays entirely on your device, under your control.
      </>
    ),
  },
  {
    title: 'Intentional Consumption',
    imgSrc: require('@site/static/img/Brain.png').default,
    description: (
      <>
        Break free from algorithmic feeds and attention-grabbing interfaces. Choose your sources, set your pace, and consume information on your terms.
      </>
    ),
  },
  {
    title: 'LLM Integration',
    imgSrc: require('@site/static/img/Chatbot.png').default,
    description: (
      <>
        Seamlessly connect with your AI assistant through Model Context Protocol. Query, summarize, and analyze your curated content using the LLM you trust.
      </>
    ),
  },
];

function Feature({title, Svg, imgSrc, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        {Svg ? (
          <Svg className={styles.featureSvg} role="img" />
        ) : imgSrc ? (
          <img src={imgSrc} alt={title} className={styles.featureSvg} />
        ) : null}
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
