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
    title: 'Privacy-First',
    imgSrc: require('@site/static/img/Privacy.png').default,
    description: (
      <>
        No accounts, no tracking, no ads. Colino works entirely offline and stores everything locally, so you stay in control.
      </>
    ),
  },
  {
    title: 'Focus, No Distraction',
    imgSrc: require('@site/static/img/Brain.png').default,
    description: (
      <>
        Colino is designed for deep focus. No notifications, no feeds, no noise—just the content you choose, when you want it.
      </>
    ),
  },
  {
    title: 'Smart Summarization',
    imgSrc: require('@site/static/img/Chatbot.png').default,
    description: (
      <>
        Colino uses advanced AI to summarize articles and videos, helping you stay informed without information overload.
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
