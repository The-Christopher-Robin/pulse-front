import type { GetServerSideProps } from 'next';
import Layout from '@/components/Layout';
import { getAssignments } from '@/lib/api';
import type { AssignmentMap } from '@/lib/experiments';

type Props = {
  userId: string;
  assignments: AssignmentMap;
};

export default function ExperimentsPage({ userId, assignments }: Props) {
  const keys = Object.keys(assignments).sort();
  return (
    <Layout title="Experiments">
      <section className="hero">
        <h1>Your live experiments</h1>
        <p>
          Each card below shows a running experiment and the variant assigned to <code>{userId}</code>.
          Bucketing is deterministic, so you will land on the same variants on every refresh until the
          experiment definition changes.
        </p>
      </section>
      <section className="grid">
        {keys.map((key) => {
          const a = assignments[key];
          return (
            <div key={key} className="card" style={{ padding: 14 }}>
              <div style={{ fontWeight: 600, marginBottom: 6 }}>{key}</div>
              <div>
                <span className="exp-chip">variant</span>
                {a.variant_key}
              </div>
              <div>
                <span className="exp-chip">exposed</span>
                {a.exposed ? 'yes' : 'no (holdout)'}
              </div>
            </div>
          );
        })}
      </section>
    </Layout>
  );
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ req }) => {
  const cookie = req.headers.cookie ?? '';
  const { assignments, user_id } = await getAssignments({ cookie });
  return { props: { assignments: assignments ?? {}, userId: user_id } };
};
