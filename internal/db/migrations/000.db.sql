-- DROP DATABASE IF EXISTS food_stock;

CREATE DATABASE food_stock
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1
    IS_TEMPLATE = False;

\c food_stock

CREATE TYPE public.locations AS ENUM
    ('fridge', 'freezer', 'pantry');

CREATE TABLE IF NOT EXISTS public.roles
(
    id integer NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1 ),
    name character varying(30) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT roles_pk PRIMARY KEY (id),
    CONSTRAINT roles_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS public.users
(
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    username character varying(50) COLLATE pg_catalog."default" NOT NULL,
    password_hash text COLLATE pg_catalog."default" NOT NULL,
    name character varying(100) COLLATE pg_catalog."default" NOT NULL,
    email text COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT users_pk PRIMARY KEY (id),
    CONSTRAINT email_unique UNIQUE (email),
    CONSTRAINT username_unique UNIQUE (username)
);

CREATE TABLE IF NOT EXISTS public.households
(
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name character varying(50) COLLATE pg_catalog."default" NOT NULL,
    address text COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT households_pk PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS public.users_households
(
    user_id uuid NOT NULL,
    household_id uuid NOT NULL,
    role_id integer NOT NULL,
    joined_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT users_households_pk PRIMARY KEY (user_id, household_id),
    CONSTRAINT households_fk FOREIGN KEY (household_id)
        REFERENCES public.households (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
        NOT VALID,
    CONSTRAINT roles_fk FOREIGN KEY (role_id)
        REFERENCES public.roles (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT
        NOT VALID,
    CONSTRAINT users_fk FOREIGN KEY (user_id)
        REFERENCES public.users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
        NOT VALID
);

CREATE TABLE IF NOT EXISTS public.product_types
(
    id integer NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1 ),
    name character varying(30) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT product_types_pk PRIMARY KEY (id),
    CONSTRAINT product_type_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS public.products
(
    id integer NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1 ),
    name character varying(30) COLLATE pg_catalog."default" NOT NULL,
    type_id integer NOT NULL,
    CONSTRAINT products_pk PRIMARY KEY (id),
    CONSTRAINT products_name_unique UNIQUE (name),
    CONSTRAINT product_types_fk FOREIGN KEY (type_id)
        REFERENCES public.product_types (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS public.products_households
(
    product_id integer NOT NULL,
    household_id uuid NOT NULL,
    location locations NOT NULL,
    needs_refill boolean NOT NULL DEFAULT false,
    refilled_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT products_households_pk PRIMARY KEY (product_id, household_id),
    CONSTRAINT households_fk FOREIGN KEY (household_id)
        REFERENCES public.households (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
        NOT VALID,
    CONSTRAINT products_fk FOREIGN KEY (product_id)
        REFERENCES public.products (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
        NOT VALID
);

CREATE TABLE IF NOT EXISTS public.invites
(
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    email text COLLATE pg_catalog."default" NOT NULL,
    household_id uuid NOT NULL,
    invited_by uuid NOT NULL,
    token_hash character(64) COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone NOT NULL DEFAULT (now() + '24:00:00'::interval),
    accepted_at timestamp with time zone,
    CONSTRAINT invites_pk PRIMARY KEY (id),
    CONSTRAINT invites_token_hash_unique UNIQUE (token_hash),
    CONSTRAINT households_fk FOREIGN KEY (household_id)
        REFERENCES public.households (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE,
    CONSTRAINT users_fk FOREIGN KEY (invited_by)
        REFERENCES public.users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);
