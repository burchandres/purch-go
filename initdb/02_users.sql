--
-- PostgreSQL database dump
--

\restrict bqLagHp47EzKcBaqYPCkaRbxhxQbkZNIqyCByPU0hmpUw8bNtvWtzUEcLfO91e6

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
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.users (id, first_name, last_name, username, password, income, income_rate) VALUES ('df901f07-7314-41b7-880c-1230038a328e', 'test', 'user', 'testuser', '$2a$10$AQOLxQTZS/V3XS/3qFkUxe3RQ6UqqOW/a2zuxJAN0Nhpe5zUVnoSu', 1000, 'weekly');
INSERT INTO public.users (id, first_name, last_name, username, password, income, income_rate) VALUES ('b723a5fc-190f-41b0-95da-8cbc014cbafa', 'foo', 'bar', 'testuser_242b6de0', '$2a$10$3JmkhJdVfOyyTVjkuIvGKeDYGs4dG/MND1N4PY0HEfs9pG5ryAb2q', 100000, 'annual');


--
-- PostgreSQL database dump complete
--

\unrestrict bqLagHp47EzKcBaqYPCkaRbxhxQbkZNIqyCByPU0hmpUw8bNtvWtzUEcLfO91e6

