-- Eliminar columna full_name de operator_profiles si existe
ALTER TABLE operator_profiles DROP COLUMN IF EXISTS full_name;
