ALTER TABLE public.users DROP COLUMN password_hash;
ALTER TABLE public.users DROP COLUMN email;

ALTER TABLE public.users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE public.users
    ADD CONSTRAINT users_auth_fk FOREIGN KEY (id) REFERENCES auth.users(id) ON DELETE CASCADE;
ALTER TABLE public.users ADD COLUMN is_active boolean NOT NULL DEFAULT true;

ALTER TABLE public.users DROP CONSTRAINT IF EXISTS username_unique;
ALTER TABLE public.users DROP COLUMN username;

CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    INSERT INTO public.users (id, name)
    VALUES (
        NEW.id,
        NEW.raw_user_meta_data ->> 'name'
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER on_auth_user_created
AFTER INSERT ON auth.users
FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();
