

import React from 'react';
import Navbar from '@theme-original/Navbar';
// ...existing code...
import NavbarItem from '@theme/NavbarItem';

export default function CustomNavbar(props) {
  const items = props.items || [];
  const githubIndex = items.findIndex(
    (item) => item.href && item.href.includes('github.com')
  );
  const leftItems = githubIndex === -1 ? items : items.slice(0, githubIndex);
  const rightItems = githubIndex === -1 ? [] : items.slice(githubIndex);

  return (
    <Navbar {...props}>
      {leftItems.map((item, idx) => (
        <NavbarItem {...item} key={item.label || idx} />
      ))}
  {/* SearchBar removed */}
      {rightItems.map((item, idx) => (
        <NavbarItem {...item} key={item.label || idx} />
      ))}
    </Navbar>
  );
}
