-- Trigger ni o'chirish
DROP TRIGGER IF EXISTS update_products_updated_at ON products;

-- Function ni o'chirish
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Table ni o'chirish
DROP TABLE IF EXISTS products;