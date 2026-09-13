import { createFileRoute } from '@tanstack/react-router';
import { RouteGuard } from '@/components/route-guard';
import RequestRewriteRulesManagement from '@/features/request-rewrite-rules';

function ProtectedRequestRewriteRules() {
  return (
    <RouteGuard requiredScopes={['read_channels']} scopeLevel="system">
      <RequestRewriteRulesManagement />
    </RouteGuard>
  );
}

export const Route = createFileRoute('/_authenticated/request-rewrite-rules/')({
  component: ProtectedRequestRewriteRules,
});
