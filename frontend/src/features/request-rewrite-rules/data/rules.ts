import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { graphqlRequest } from '@/gql/graphql';
import { useErrorHandler } from '@/hooks/use-error-handler';
import {
  CreateRequestRewriteRuleInput,
  RequestRewriteRule,
  RequestRewriteRuleConnection,
  UpdateRequestRewriteRuleInput,
  requestRewriteRuleConnectionSchema,
  requestRewriteRuleSchema,
} from './schema';

const FIELD_MAPS_SELECTION = `
  fieldMaps {
    path
    values {
      from
      to
    }
  }
`;

const RULES_QUERY = `
  query GetRequestRewriteRules(
    $first: Int
    $after: Cursor
    $last: Int
    $before: Cursor
    $where: RequestRewriteRuleWhereInput
    $orderBy: RequestRewriteRuleOrder
  ) {
    requestRewriteRules(first: $first, after: $after, last: $last, before: $before, where: $where, orderBy: $orderBy) {
      edges {
        node {
          id
          createdAt
          updatedAt
          name
          description
          modelPattern
          status
          ${FIELD_MAPS_SELECTION}
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
      totalCount
    }
  }
`;

const CREATE_RULE_MUTATION = `
  mutation CreateRequestRewriteRule($input: CreateRequestRewriteRuleInput!) {
    createRequestRewriteRule(input: $input) {
      id
      createdAt
      updatedAt
      name
      description
      modelPattern
      status
      ${FIELD_MAPS_SELECTION}
    }
  }
`;

const UPDATE_RULE_MUTATION = `
  mutation UpdateRequestRewriteRule($id: ID!, $input: UpdateRequestRewriteRuleInput!) {
    updateRequestRewriteRule(id: $id, input: $input) {
      id
      createdAt
      updatedAt
      name
      description
      modelPattern
      status
      ${FIELD_MAPS_SELECTION}
    }
  }
`;

const DELETE_RULE_MUTATION = `
  mutation DeleteRequestRewriteRule($id: ID!) {
    deleteRequestRewriteRule(id: $id)
  }
`;

const UPDATE_RULE_STATUS_MUTATION = `
  mutation UpdateRequestRewriteRuleStatus($id: ID!, $status: RequestRewriteRuleStatus!) {
    updateRequestRewriteRuleStatus(id: $id, status: $status)
  }
`;

const BULK_DELETE_RULES_MUTATION = `
  mutation BulkDeleteRequestRewriteRules($ids: [ID!]!) {
    bulkDeleteRequestRewriteRules(ids: $ids)
  }
`;

const BULK_ENABLE_RULES_MUTATION = `
  mutation BulkEnableRequestRewriteRules($ids: [ID!]!) {
    bulkEnableRequestRewriteRules(ids: $ids)
  }
`;

const BULK_DISABLE_RULES_MUTATION = `
  mutation BulkDisableRequestRewriteRules($ids: [ID!]!) {
    bulkDisableRequestRewriteRules(ids: $ids)
  }
`;

interface QueryRulesArgs {
  first?: number;
  after?: string;
  last?: number;
  before?: string;
  where?: Record<string, any>;
  orderBy?: {
    field: 'CREATED_AT' | 'UPDATED_AT';
    direction: 'ASC' | 'DESC';
  };
}

export function useQueryRequestRewriteRules(args: QueryRulesArgs) {
  return useQuery({
    queryKey: ['request-rewrite-rules', args],
    queryFn: async () => {
      const data = await graphqlRequest<{ requestRewriteRules: RequestRewriteRuleConnection }>(RULES_QUERY, args);
      return requestRewriteRuleConnectionSchema.parse(data.requestRewriteRules);
    },
  });
}

export function useCreateRequestRewriteRule() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (input: CreateRequestRewriteRuleInput) => {
      try {
        const data = await graphqlRequest<{ createRequestRewriteRule: RequestRewriteRule }>(CREATE_RULE_MUTATION, { input });
        return requestRewriteRuleSchema.parse(data.createRequestRewriteRule);
      } catch (error) {
        handleError(error, { context: t('requestRewriteRules.dialogs.create.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.createSuccess'));
    },
  });
}

export function useUpdateRequestRewriteRule() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: UpdateRequestRewriteRuleInput }) => {
      try {
        const data = await graphqlRequest<{ updateRequestRewriteRule: RequestRewriteRule }>(UPDATE_RULE_MUTATION, { id, input });
        return requestRewriteRuleSchema.parse(data.updateRequestRewriteRule);
      } catch (error) {
        handleError(error, { context: t('requestRewriteRules.dialogs.edit.title') });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.updateSuccess'));
    },
  });
}

export function useDeleteRequestRewriteRule() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (id: string) => {
      try {
        await graphqlRequest(DELETE_RULE_MUTATION, { id });
      } catch (error) {
        handleError(error, { context: 'Delete Request Rewrite Rule' });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.deleteSuccess'));
    },
  });
}

export function useUpdateRequestRewriteRuleStatus() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async ({ id, status }: { id: string; status: 'enabled' | 'disabled' }) => {
      try {
        await graphqlRequest(UPDATE_RULE_STATUS_MUTATION, { id, status });
      } catch (error) {
        handleError(error, { context: 'Update Request Rewrite Rule Status' });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.statusUpdateSuccess'));
    },
  });
}

export function useBulkDeleteRequestRewriteRules() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (ids: string[]) => {
      try {
        await graphqlRequest(BULK_DELETE_RULES_MUTATION, { ids });
      } catch (error) {
        handleError(error, { context: 'Bulk Delete Request Rewrite Rules' });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.bulkDeleteSuccess'));
    },
  });
}

export function useBulkEnableRequestRewriteRules() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (ids: string[]) => {
      try {
        await graphqlRequest(BULK_ENABLE_RULES_MUTATION, { ids });
      } catch (error) {
        handleError(error, { context: 'Bulk Enable Request Rewrite Rules' });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.bulkEnableSuccess'));
    },
  });
}

export function useBulkDisableRequestRewriteRules() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { handleError } = useErrorHandler();

  return useMutation({
    mutationFn: async (ids: string[]) => {
      try {
        await graphqlRequest(BULK_DISABLE_RULES_MUTATION, { ids });
      } catch (error) {
        handleError(error, { context: 'Bulk Disable Request Rewrite Rules' });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['request-rewrite-rules'] });
      toast.success(t('requestRewriteRules.messages.bulkDisableSuccess'));
    },
  });
}
