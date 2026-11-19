import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];

export type SalesReport = Schemas['SalesReport'];
export type SalesReportGroup = Schemas['SalesReportGroup'];
export type ProductsReport = Schemas['ProductsReport'];
export type ProductsReportSort = Schemas['ProductsReportSort'];
export type SaleProfitsReport = Schemas['SaleProfitsReport'];
export type ProfitFigures = Schemas['ProfitFigures'];
export type IdleStockReport = Schemas['IdleStockReport'];
export type CashClosingsReport = Schemas['CashClosingsReport'];
export type CashClosingsReportGroup = Schemas['CashClosingsReportGroup'];
export type TeamControlReport = Schemas['TeamControlReport'];
export type TeamControlFigures = Schemas['TeamControlFigures'];
export type ReportBucket = Schemas['ReportBucket'];
export type StockAdjustmentsReport = Schemas['StockAdjustmentsReport'];
export type StockAdjustmentsReportGroup = Schemas['StockAdjustmentsReportGroup'];
export type ReportKind = Schemas['ReportKind'];
export type ReportFormat = Schemas['ReportFormat'];
