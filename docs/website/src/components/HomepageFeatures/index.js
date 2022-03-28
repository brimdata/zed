import React from "react";
import clsx from "clsx";
import styles from "./styles.module.css";

const FeatureList = [
  {
    title: "Effortless Data Discovery",

    description: (
      <>Quickly see the types in your data and get a sample of each.</>
    ),
  },
  {
    title: "Incredible Tooling",

    description: (
      <>
        Scale down to your desktop with tools like <code>zq</code>, then scale
        up to the cloud with our <code>zed lake</code>.
      </>
    ),
  },
  {
    title: "Super Structured Data",

    description: (
      <>
        Enjoy the benefits of super-structured data. No more rigid schemas. No
        more JSON strings.
      </>
    ),
  },
];

function Feature({ Svg, title, description }) {
  return (
    <div className={clsx("col col--4")}>
      <div className="padding-horiz--md">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures() {
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
