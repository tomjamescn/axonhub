import { z } from 'zod';

export const requestRewriteValueMapSchema = z.object({
  from: z.string(),
  to: z.string(),
});

export const requestRewriteFieldMapSchema = z.object({
  path: z.string(),
  values: z.array(requestRewriteValueMapSchema).default([]),
});

export const requestRewriteRuleSchema = z.object({
  id: z.string(),
  createdAt: z.coerce.date(),
  updatedAt: z.coerce.date(),
  userID: z.number().nullable().optional(),
  name: z.string(),
  description: z.string(),
  modelPattern: z.string(),
  status: z.enum(['enabled', 'disabled', 'archived']),
  fieldMaps: z.array(requestRewriteFieldMapSchema).default([]),
});

export const requestRewriteRuleEdgeSchema = z.object({
  node: requestRewriteRuleSchema,
  cursor: z.string(),
});

export const requestRewriteRulePageInfoSchema = z.object({
  hasNextPage: z.boolean(),
  hasPreviousPage: z.boolean(),
  startCursor: z.string().nullable(),
  endCursor: z.string().nullable(),
});

export const requestRewriteRuleConnectionSchema = z.object({
  edges: z.array(requestRewriteRuleEdgeSchema),
  pageInfo: requestRewriteRulePageInfoSchema,
  totalCount: z.number(),
});

export type RequestRewriteValueMap = z.infer<typeof requestRewriteValueMapSchema>;
export type RequestRewriteFieldMap = z.infer<typeof requestRewriteFieldMapSchema>;
export type RequestRewriteRule = z.infer<typeof requestRewriteRuleSchema>;
export type RequestRewriteRuleConnection = z.infer<typeof requestRewriteRuleConnectionSchema>;

export interface CreateRequestRewriteRuleInput {
  name: string;
  description?: string;
  modelPattern: string;
  fieldMaps: RequestRewriteFieldMap[];
}

export interface UpdateRequestRewriteRuleInput {
  name?: string;
  description?: string;
  modelPattern?: string;
  fieldMaps?: RequestRewriteFieldMap[];
}
