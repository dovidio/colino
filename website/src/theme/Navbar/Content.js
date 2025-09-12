import React from 'react';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import { useThemeConfig } from '@docusaurus/theme-common';
import Link from '@docusaurus/Link';
import useBaseUrl from '@docusaurus/useBaseUrl';
import NavbarItem from '@theme/NavbarItem';
// ...existing code...

export default function NavbarContent(props) {
  const {siteConfig} = useDocusaurusContext();
  const {
    navbar: {title, logo, items = []} = {},
  } = useThemeConfig();

  // Split items by position (default: left)
  const leftItems = items.filter((item) => item.position !== 'right');
  const rightItems = items.filter((item) => item.position === 'right');

  return (
    <>
      <div className="navbar__brand">
        {logo && (
          <Link className="navbar__logo" to={useBaseUrl('/')}>
            <img src={useBaseUrl(logo.src)} alt={logo.alt || title} />
          </Link>
        )}
        {title && (
          <Link className="navbar__title" to={useBaseUrl('/')}>
            {title}
          </Link>
        )}
      </div>
      <div className="navbar__items">
        {leftItems.map((item, idx) => (
          <NavbarItem {...item} key={item.label || idx} />
        ))}
      </div>
      <div className="navbar__items navbar__items--right">
        {rightItems.map((item, idx) => (
          <NavbarItem {...item} key={item.label || idx} />
        ))}
      </div>
    </>
  );
}
