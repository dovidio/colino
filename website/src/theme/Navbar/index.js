import React from 'react';
import NavbarLayout from '@theme-original/Navbar/Layout';
import NavbarContent from '@theme/Navbar/Content';

export default function Navbar(props) {
  return (
    <NavbarLayout>
      <NavbarContent {...props} />
    </NavbarLayout>
  );
}
