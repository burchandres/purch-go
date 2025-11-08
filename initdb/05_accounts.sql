--
-- PostgreSQL database dump
--

\restrict ERSFsn6aYOGO7dr6T4t1t7s8erLfmXJCSfCgyITqNbgQtzhcrwQXsHh7UIyC5si

-- Dumped from database version 18.0
-- Dumped by pg_dump version 18.0 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.accounts (id, item_id, name, available_balance, current_balance, type, sub_type) VALUES ('C1-id-checking', 'C1-id', 'Young Adult Checking', 100, 150, 'Depository', 'Checking');
INSERT INTO public.accounts (id, item_id, name, available_balance, current_balance, type, sub_type) VALUES ('WF-id-credit-card', 'WF-id', 'Active Cash Credit Card', 100, 900, 'Credit', '');
INSERT INTO public.accounts (id, item_id, name, available_balance, current_balance, type, sub_type) VALUES ('xMMbBmNyv1CmDq6VRkWbT13Ao8mJwESyE3vxN', 'l33MWowj6xF91qWEeZrMho5XME16vBiZLae7e', 'Checking', NULL, NULL, '', NULL);


--
-- PostgreSQL database dump complete
--

\unrestrict ERSFsn6aYOGO7dr6T4t1t7s8erLfmXJCSfCgyITqNbgQtzhcrwQXsHh7UIyC5si

