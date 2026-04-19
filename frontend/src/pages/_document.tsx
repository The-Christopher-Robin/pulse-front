import { Html, Head, Main, NextScript } from 'next/document';

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        <meta name="description" content="PulseFront: a full-stack customer-facing platform with Go backend, SSR, and A/B experiments." />
      </Head>
      <body>
        <Main />
        <NextScript />
      </body>
    </Html>
  );
}
