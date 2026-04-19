import Head from 'next/head';
import Link from 'next/link';
import type { ReactNode } from 'react';

type Props = {
  title: string;
  children: ReactNode;
};

export default function Layout({ title, children }: Props) {
  return (
    <>
      <Head>
        <title>{title} | PulseFront</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      <header className="nav">
        <div className="nav-inner">
          <Link href="/" className="logo">PulseFront</Link>
          <nav>
            <Link href="/" style={{ marginRight: 16 }}>Shop</Link>
            <Link href="/experiments">Experiments</Link>
          </nav>
        </div>
      </header>
      <main className="container">{children}</main>
    </>
  );
}
